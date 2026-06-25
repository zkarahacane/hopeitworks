# ADR — Modèle d'environnement des agents : la plateforme provisionne, l'agent consomme

> **Statut** : Accepté — validé empiriquement le 2026-06-25 (voir §8). Amendé le 2026-06-25
> (tests userns §8.5–8.6 ; build-verification §3.5 ; état code réel §9 ; périmètre Docker-first).
> **Supersède la direction « substrat durci microVM en premier »** :
> `agent-substrate-abstraction-adr.md` (décision #3, « microsandbox = prod default »)
> et `agent-runtime-capabilities-plan.md` (décision #14, « build first = microsandbox »).
> Le port `AgentRuntime` et l'abstraction restent valables ; c'est le **choix de substrat
> par défaut** et la **raison d'être** du substrat durci qui changent.
> **Date** : 2026-06-25.

## 1. Contexte

Un agent a besoin d'un **environnement de dev complet et jetable** : être root chez lui,
installer des paquets, builder, lancer des tests, et disposer de **dépendances** (une DB,
un redis, un service mock…). Trois contraintes produit, non négociables :

1. **« Jean-Michel Random »** clone le repo depuis GitHub et veut l'essayer sur son poste
   en deux minutes, sans monter d'infra → **Docker** (marche sur Mac/Windows/Linux).
2. **Clients en production sur OpenShift/Kubernetes** → tout ce qui touche la prod doit
   être **Kube-natif, scalable, industrialisable**.
3. **Un seul produit, portable** : le même code doit tourner du laptop à OpenShift, le
   substrat s'adaptant à la plateforme (rôle de `port.AgentRuntime`).

Historiquement, on était parti sur un **substrat microVM durci (microsandbox / libkrun)**
pour donner aux agents du **DinD/testcontainers natif** avec isolation noyau. Cet ADR
corrige cette direction.

## 2. Le constat qui retourne le problème

En décortiquant ce qu'un agent veut *réellement* faire, **Docker nu ne manque d'aucune
capacité fonctionnelle** :

| Besoin de l'agent | Couvert par Docker nu ? |
|---|---|
| installer paquets, builder, tests unitaires (root dans son conteneur) | ✅ |
| une **DB de dev** | ✅ sidecar sur le réseau Docker (déjà fait, P2c) |
| **builder une image** | ✅ buildah/kaniko/buildkit (sans daemon) sur hôte permissif — ⚠️ sous OpenShift `restricted`, policy de build dédiée requise (§3.5) |
| **faire tourner des conteneurs** (testcontainers, `docker run`) | ✅ via socket de l'hôte (Docker-out-of-Docker) ou `--privileged` |

**La seule chose que Docker nu ne sait pas faire, c'est le faire *en sécurité*.** Chaque
manière d'accorder le DinD à l'agent lui ouvre une **porte vers l'hôte** :
- monter le socket → l'agent pilote le **daemon de l'hôte** (monte `/`, lance un conteneur
  privileged, sort) ;
- `--privileged` → root hôte de fait.

Donc le problème n'a **jamais** été « quel substrat » mais **« qui opère les conteneurs »**.

## 3. Décision — l'inversion

**La plateforme est l'opérateur d'infrastructure ; l'agent est un simple consommateur.**
L'agent ne lance **jamais** de conteneur. Conséquence directe : il n'a jamais besoin d'un
daemon Docker, ni du socket, ni de `--privileged` → **un conteneur non-privilégié suffit,
partout.**

Mécanique :

1. L'objet **`environment`** déclare les dépendances (`services[]` : postgres, redis…).
2. La plateforme **déploie ces conteneurs** *avant* le lancement de l'agent — sidecars sur
   le réseau Docker (substrat `docker`) ou containers-in-Pod (substrat `kubernetes`).
3. La plateforme **injecte les conn-strings** au démarrage de l'agent (env vars / bundle
   fetch-at-startup, P1).
4. L'agent tourne **non-privilégié, sans socket** ; il n'a accès qu'au **réseau qu'on lui
   donne** : ses services déclarés + une **allowlist egress** (pour taper npm/pypi/crates).
   Le reste est coupé (réseau Docker dédié + règles egress / NetworkPolicy).

**Un seul déclencheur : l'environnement est défini en amont et approuvé par le user.** L'env
liste `postgres` → la plateforme le déploie *avant* le lancement → l'agent lit `DATABASE_URL`.
**L'agent ne peut pas ajouter de dépendance lui-même** : s'il lui en faut une non déclarée,
c'est un **signal au user pour réviser l'environnement**, pas une action self-service.

> **Décision (2026-06-25)** : **pas de provisioning dynamique par l'agent** (type
> `provision_dependency`). L'environnement est un **contrat figé, approuvé par l'humain** ;
> l'agent **consomme, il n'étend jamais**. On garde le **contrôle humain sur l'infra** (et,
> accessoirement, on supprime la surface de sécu quota/DoS/catalogue). *(MCP reste utilisé
> pour les autres capacités — skills, tools, serveurs MCP métier ; ce n'est que le tool de
> provisioning dynamique qui est écarté.)*

### Cas résiduel — quand le *code de l'agent* appelle Docker

Si le code à exécuter attend lui-même un endpoint Docker (un test `testcontainers-go`,
`docker compose up`), on **n'accorde jamais de DinD**. On pointe son `DOCKER_HOST` vers un
**broker contrôlé par la plateforme** :
- sur **Kubernetes** → **KubeDock** (traduit les appels Docker API en vrais Pods pilotés
  par la plateforme) ;
- sur **Docker** → socket-proxy filtré, ou un `dockerd` éphémère dédié au run.

Même principe : l'agent ne touche jamais le daemon de l'hôte, il parle à un endpoint que
**la plateforme** possède.

> **Validé empiriquement (§8.3–8.6)** : sur OpenShift en SCC restricted, **ni** le DinD
> privileged (rejeté par l'admission PodSecurity) **ni** le DinD rootless (RootlessKit
> échoue, **y compris avec userns GA** — §8.6) ne passent. Le **broker** (KubeDock) ou
> **Kata** (privileged-dind sûr dans un microVM) sont les seules voies pour ce cas résiduel
> sur OpenShift.

### 3.5 Build-verification — « est-ce que ça build ? »

Pour la plupart des stacks, « ça build » = `go build` / `npm run build` / `mvn package` :
**juste la toolchain dans le conteneur de l'agent, zéro Docker**. Le `docker build` n'entre
en jeu que si le livrable est une *image*. Dans ce cas on build **sans daemon ni privileged**
via **buildah / kaniko / BuildKit rootless** — et **les images de release restent à la CI**
(l'agent ne fait qu'un check pass/fail, il ne publie rien).

- **Hôte permissif** (poste, nœud Linux normal) → ✅ trivial.
- **OpenShift `restricted` stock** → ❌ le seccomp `RuntimeDefault` bloque
  `unshare(CLONE_NEWUSER)` dont le builder rootless a besoin (validé §8.5). Il faut une
  **policy de build dédiée** (seccomp Localhost autorisant `CLONE_NEWUSER` + plage subuid),
  p.ex. via **OpenShift Builds / Shipwright**. **Pas privileged**, mais **pas le `restricted`
  stock non plus**.
- ⚠️ **userns (`hostUsers:false`) ne lève PAS ce blocage** (validé §8.5) : il sécurise le pod
  mais n'autorise pas le tooling rootless imbriqué sous `restricted`.

## 4. Conséquences

- **Un conteneur Docker nu non-privilégié suffit partout** (Mac + OpenShift, code
  identique). La question du substrat durci **se dissout** pour le cas par défaut.
- **L'isolation devient optionnelle**, gouvernée par le **modèle de menace** (§6) — ce
  n'est plus une caractéristique du produit mais une *policy d'adapter*.
- **Réutilise massivement l'existant** : `environment` / sidecars / conn-strings (P2c),
  fetch-at-startup (P1), capacités MCP. **Nouveau** : scoping réseau+egress par run, broker
  pour le cas testcontainers (différé).
- **microsandbox sort du chemin critique** (voir §5.1).

## 5. Alternatives considérées et pourquoi écartées

### 5.1 microsandbox (libkrun microVM) — **ÉCARTÉ**

- **Son unique différenciateur (DinD natif) ne fonctionne pas.** Spike du 2026-06-25 (VM
  Lima, `/dev/kvm` OK) : `dockerd` ne démarre proprement qu'en **avant-plan ~12 s** avec
  des flags non-défaut (`--storage-driver=vfs --iptables=false --bridge=none`) ; en
  arrière-plan ou dès qu'on sollicite vraiment le daemon (créer un conteneur), **le
  démarrage du daemon tue la session du guest** (le setup overlay/iptables casse le relay
  de l'agent microsandbox). La config Docker par défaut casse instantanément.
- **Beta v0.5.8**, SDK Go (cgo), **KVM-only** → sur Mac, VM Lima lourde (~6 GiB) :
  **casse l'exigence « Jean-Michel clone & run »**.
- **Ne sert aucun des deux mondes** : ni zéro-infra-laptop, ni le chemin prod (en prod =
  OpenShift, où le microVM mature est **Kata**, pas microsandbox).
- **Sous l'inversion (§3), l'agent ne lance plus de conteneurs** → un vrai kernel par
  agent ne sert plus à rien.
- → Code **conservé derrière `//go:build microsandbox`** (inerte, build par défaut
  inchangé), **retiré du chemin critique**. Réactivable si jamais un besoin le justifie.

### 5.2 Kata / OpenShift sandboxed containers (microVM mature) — **non retenu (pas rejeté sur le fond)**

- C'est le **bon** outil microVM pour OpenShift *si* on avait besoin d'iso-VM + DinD natif,
  et il est **supporté Red Hat**. La version robuste de ce que microsandbox tentait.
- Mais **l'inversion supprime le besoin de lancer des conteneurs dans l'agent** → plus
  besoin d'un vrai kernel par agent. Kata n'apporterait qu'une isolation *qu'on peut
  obtenir plus simplement* (§5.3) quand elle est nécessaire.
- **Exige des nœuds avec virtualisation (KVM)** → pas universel.
- → **Hors chemin critique.** Reste l'option de référence si un besoin
  *hard-multi-tenant + DinD-réel-pour-le-code-agent* émerge un jour.

### 5.3 gVisor (RuntimeClass) — **conservé comme seule option de durcissement, conditionnelle**

- **N'ajoute aucune capacité fonctionnelle** ; ajoute une **frontière d'isolation** sans
  VM ni KVM (kernel userspace, OpenShift-viable).
- Pas de DinD — **sans objet ici**, puisque l'agent ne lance plus de conteneurs.
- **Non requis pour le cas par défaut** (code de confiance). Activé *uniquement* face à du
  code non-fiable (§6), sur le conteneur **consommateur** — surface minuscule à isoler.

### 5.4 DinD `--privileged` / montage du socket hôte sur Docker nu — **ÉCARTÉ comme défaut**

- **Fonctionne** (et c'est acceptable sur ta propre machine : ta machine, ton risque).
- Mais = **porte d'évasion vers l'hôte** → inacceptable en multi-tenant / code non-fiable.
- C'est précisément ce que l'inversion évite. Quand le code de l'agent a *vraiment* besoin
  de Docker, le **broker** (§3, KubeDock/proxy) le remplace.

### 5.5 Tout exécuter dans une VM Lima (sur Mac) — **ÉCARTÉ**

- Build Go CGO en virtualisation imbriquée (lent), ~6 GiB en permanence, **diverge de la
  CI** (`SUBSTRATE=docker`), pas de front, **casse l'UX d'essai local**.

## 6. Modèle de menace — la décision qui gate l'isolation

L'isolation n'est plus un choix de substrat mais une réponse au modèle de menace :

- **Code de confiance / single-tenant** (le cas par défaut : tes propres agents ; un client
  qui déploie *ses* agents sur *son* cluster pour *son* code) → **Docker partout, terminé.**
  Aucun substrat durci nécessaire.
- **Code non-fiable / multi-tenant** (images communautaires, code généré arbitraire,
  plusieurs tenants sur les mêmes nœuds) → ajouter **gVisor** (RuntimeClass) sur le
  conteneur consommateur, *pour ces workloads-là uniquement*. **Jamais besoin de microVM** —
  plus personne ne fait de nested containers dans l'agent.

## 7. Portefeuille de substrats résultant

| Plateforme / usage | `SUBSTRATE` | Isolation | Notes |
|---|---|---|---|
| Jean-Michel, essai local (Mac/Win/Linux) | `docker` | conteneur nu | défaut, déjà fait |
| Client OpenShift/K8s, code de confiance | `kubernetes` (runc) | conteneur nu | Pod par run, sidecars-in-Pod |
| Client OpenShift/K8s, code non-fiable | `kubernetes` + RuntimeClass `gvisor` | userspace kernel | + broker (KubeDock) si testcontainers |

**Deux adapters suffisent** : `docker` (partout) + `kubernetes` (prod, RuntimeClass =
bouton d'isolation). microsandbox parqué (§5.1).

## 8. Validation empirique (2026-06-25)

1. **Spike DinD microsandbox** — sur VM Lima avec `/dev/kvm` : kernel invité 6.12.68,
   cgroup v2, overlay, full caps présents ; `docker:dind` boote ; mais **`dockerd` ne tient
   pas** (tue la session du guest dès qu'on l'utilise réellement). → §5.1. **Échec.**
2. **Test de l'inversion** — sur Docker nu (Mac) : la « plateforme » crée un réseau de run
   et provisionne un `postgres` ; un conteneur **agent `uid=1000`, sans binaire docker, sans
   socket**, reçoit `DATABASE_URL` injectée et **consomme la DB** (`CREATE TABLE` / `INSERT`
   / `SELECT count = 1`). → **Succès.** Prouve qu'un agent non-privilégié sans aucun accès
   Docker obtient un env de dev avec dépendances, sur la plateforme la plus simple qui soit.
3. **DinD sur K8s (kind v1.36, privileged)** — pod `docker:dind` privileged : dockerd
   29.6.0 + **overlayfs** + cgroup v2 ; conteneur imbriqué lancé (`alpine`). → **Marche
   mécaniquement** sur K8s (contraste net avec microsandbox §5.1) — **mais privileged**
   (porte d'évasion vers le nœud).
4. **DinD sous OpenShift-restricted (kind, namespace PSS `restricted`)** — proxy fidèle de
   la SCC restricted d'OpenShift :
   - **privileged → rejeté par l'admission** (PodSecurity : privileged / allowPrivilegeEscalation
     / capabilities / runAsNonRoot / seccomp) ;
   - **rootless conforme restricted → admis mais dockerd ne démarre pas** :
     `rootlesskit … fork/exec /proc/self/exe: operation not permitted` (no-new-privs + drop
     ALL caps + seccomp RuntimeDefault empêchent la création du user namespace).
   → **Sur OpenShift sécurisé par défaut, le DinD-dans-l'agent est inatteignable** sans
   relâcher la SCC (privileged = évasion), passer par **Kata** (privileged-dind sûr dans un
   microVM), ou **KubeDock** (pas de DinD). Confirme §3 (cas résiduel) et §5.2 (Kata).
5. **Build rootless sous restricted + userns (k3s sur vrai nœud)** — pod buildah
   `runAsNonRoot` + drop ALL caps + `hostUsers:false` (userns **confirmé actif** : `uid_map`
   non-identité `0 3365994496 65536`) : **échec** `unshare(CLONE_NEWUSER): operation not
   permitted` (seccomp `RuntimeDefault`). En relâchant **uniquement** le seccomp : passe le
   `CLONE_NEWUSER` mais **échoue ensuite** (`single mapping` + `remount /: permission denied`).
   → **le build rootless ne passe pas le `restricted` stock** → policy de build dédiée (§3.5).
6. **DinD rootless sous restricted + userns (k3s)** — `docker:dind-rootless`, `hostUsers:false` :
   **échec identique au point 4** (`rootlesskit … operation not permitted`). → **userns
   améliore la sécurité du pod mais n'active PAS le tooling rootless imbriqué sous restricted.**

> **Correction de méthode** : l'agent critique avait affirmé que « userns GA (K8s 1.36)
> invalide le point 4 ». Les tests 5–6 le **réfutent empiriquement** : userns ne lève ni le
> blocage seccomp `CLONE_NEWUSER`, ni la limite de mapping subuid. Le point 4 tient.
> *(Leçon récurrente : tester, ne pas concéder une hypothèse non vérifiée.)*

## 9. État de l'implémentation (audit code 2026-06-25)

**L'inversion est déjà ~80 % en place côté Docker** — et **aucun code n'accorde de socket ni
de `--privileged` à l'agent** (`privileged=false` en dur, pas de bind socket) : l'inversion
est **structurellement déjà respectée**.

- **KEEP (fondation saine)** : port `AgentRuntime` (agnostique), adapter `docker`,
  **réseau-par-run + sidecars + conn-strings + GC**, catalogue `stacks`, capacités/MCP,
  token/cost/callback agnostiques, `CancelRun` substrate-aware. **fetch-at-startup largement
  implémenté** (`bundle_service.go` + handler + token-binding) — *pas* « à finir ».
- **THROW** : infra microsandbox — suppression franche de `Dockerfile.microsandbox`,
  `cmd/microsandbox-smoke`, `docker-compose.microsandbox.yml`, `deploy/lima/*` ; archivage du
  plan P3 ; **parking build-tag** de l'adapter Go + SDK `go.mod` (interdépendants à la
  compilation → traiter en **un seul lot**).
- **MISSING** : adapter `kubernetes` (L, **différé** — voir périmètre) ; **reaping
  substrate-aware** (M, **point dur** : les 3 reapers sont Docker-only) ; **egress allowlist
  par run** (M) ; champ config `RuntimeClass` (S) ; broker testcontainers (L, dernier).
  *(provisioning dynamique `provision_dependency` — **écarté**, voir §3 : env = contrat figé.)*
- **Pièges** : (a) `run_steps.container_id` est générique mais les reapers le comparent aux
  conteneurs **Docker** → un Pod K8s serait un faux-orphelin ; (b) `selectSubstrate` renvoie
  `nil` pour Docker → **double chemin d'exécution Docker** (legacy + via-runtime), à unifier.

### 9.1 Périmètre court terme : **Docker-first**

L'adapter `kubernetes` est **mis de côté pour l'instant**. Focus : durcir et compléter le
chemin **Docker**. Séquence non-cassante : (1) **unifier le chemin Docker** (injecter un
`docker.Runtime` non-nil au lieu de `nil`) ; (2) **egress allowlist par run** ; (3) **cleanup
microsandbox** (un seul lot). L'adapter K8s + reaping agnostique + `RuntimeClass` viendront
quand le chemin OpenShift sera priorisé.

## 10. Liens

- Corrige : `docs/agent-substrate-abstraction-adr.md` (décision #3),
  `docs/agent-runtime-capabilities-plan.md` (décision #14),
  `docs/agent-runtime-p3-microsandbox-plan.md`.
- Réutilise : modèle environment/sidecars (P2c), capacités MCP, invariant Pod-exprimable.
