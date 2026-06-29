# hopeitworks — Vision Produit

*Maintenu par François. Dernière mise à jour : 2026-06-24.*

## Vision

**hopeitworks** orchestre des agents IA pour automatiser le développement logiciel. On lance un epic, les agents codent en parallèle, corrigent leurs erreurs, et livrent des PR prêtes à merger.

**North Star** : "Hopeitworks builds itself" — la plateforme développe son propre code.

## Personas

### Zakari — Power User
Dev senior qui orchestre les agents. Il lance des epics entiers, suit l'exécution en temps réel, intervient sur les gates humaines. Il veut que les agents travaillent pendant qu'il fait autre chose.

### Karim — Dev Collègue
Dev backend qui découvre l'outil. Il connecte son repo, lance sa première story, voit le PR apparaître. Il veut gagner du temps sur le code mécanique (refactoring, boilerplate).

### Sophie — Fonctionnelle
PM/PO qui suit l'avancement du projet. Elle consulte le dashboard, approuve les changements qu'elle comprend (wording UI, config), écrit les prochaines stories en markdown. Pas besoin de coder.

### Admin
Configure l'instance : providers Git, modèles IA, pipelines, notifications, budgets. Invite des utilisateurs. Surveille les coûts globaux.

## Métriques de succès

| Métrique | Cible |
|----------|-------|
| Taux de succès auto (epic 8+ stories) | ≥ 80% |
| Temps moyen par story (implement → merge) | < 10 min |
| Coût moyen par story | < $3 |
| Containers zombies après cleanup | 0 |

## Modèle domaine

```
Project → Epic → Story → Run → Step
                           ↓
                         Agent (exec d'un harness sur un substrat isolé)
```

- **Project** : un repo Git avec sa configuration (pipeline, provider, budget)
- **Epic** : un lot de stories liées, avec un graphe de dépendances (DAG)
- **Story** : une user story avec des critères d'acceptation testables
- **Run** : une exécution complète de la pipeline pour une story
- **Step** : une étape de la pipeline (code, CI, review, merge...)
- **Agent** : un agent IA qui exécute une étape — fondamentalement « exec d'un harness » (clone repo → `agent-runtime` → CLI claude/opencode → callback HTTP) sur un **substrat d'exécution isolé et pluggable** (Docker, microVM, …)

## Capacités livrées

### Authentification et utilisateurs
- Login/logout avec JWT, inscription, reset de mot de passe
- Gestion des utilisateurs (admin), rôles admin/user
- Clés API personnelles (chiffrées AES-256)

### Projets
- Créer, configurer et gérer des projets connectés à un repo Git
- Inviter des membres, gérer les permissions par projet

### Stories et Epics
- Board dont les colonnes sont **dérivées des stages du pipeline** du projet, avec un toggle de vue :
  - **Macro** : le cycle de vie (Backlog → Running → In Review → Done → Failed)
  - **Détail** : une colonne par stage du pipeline (dans l'ordre), encadrée par les voies Backlog et Done/Failed
  - Une carte se place dans son `current_stage`, avancé en temps réel par le runtime
- Sélecteur d'epic et filtrage des stories
- Import de stories et d'epics via les connecteurs de planning (voir ci-dessous)
- Éditeur de stories (création et édition dans l'UI)
- Epics avec calcul de DAG et visualisation des dépendances
- Lancement d'un epic entier avec exécution parallèle

### Planning connectors

**Principe** : la plateforme est une couche d'exécution agnostique au planning. Les connecteurs permettent d'importer un plan externe (epics + stories) dans la base interne. Pour GitHub Projects v2, un connecteur persisté permet en plus de **renvoyer les transitions de statut** vers le tracker (write-back optionnel).

#### Sources disponibles en v1

Deux sources **input-only** :

1. **Markdown générique** — fichier `.md` avec frontmatter YAML par bloc (`key`, `epic`, `status`, `scope`, `depends_on`) + titre H1. Ce n'est PAS le format BMAD riche : il s'agit du format frontmatter minimal existant (`parser.go`).
2. **GitHub Projects v2** — board lu via GraphQL avec un PAT (classic avec `read:project`, ou fine-grained pour les projets d'organisation). L'URL du projet est saisie dans le dialog ; le token vient de la variable d'env configurée sur le projet (jamais saisi dans l'UI).

#### Modèle de données : provenance par item

Chaque story et epic porte quatre champs de provenance :

| Champ | Rôle |
|---|---|
| `source` | `manual` \| `markdown` \| `github_projects` |
| `external_id` | markdown : la `key` ; github : le node_id opaque de l'issue (jamais le numéro) |
| `source_url` | markdown : vide ; github : URL de l'issue/PR |
| `synced_at` | horodatage du dernier import ayant touché la ligne |

Le badge de provenance (`SourceBadge`) remplace l'ancienne heuristique `git_provider` de `BoardView`. Il affiche : **In-app** (manual), **Markdown** (markdown), **GitHub Projects** (github_projects). Quand `source_url` est renseignée, le badge est un lien vers l'item d'origine.

#### Import inbound

- L'import est **inbound** : la plateforme lit la source, normalise, upsert. Aucune écriture retour vers le fichier Markdown.
- Le board est une **projection générée** du plan importé, pas un miroir live.
- Pour GitHub Projects v2, un **write-back de statut optionnel** peut renvoyer chaque transition de pipeline vers le tracker (voir section ci-dessous).

#### Re-import réconciliatoire (jamais destructif)

- Re-importer réconcilie (crée / met à jour) — ne supprime jamais un item. Un item supprimé en amont est conservé (il est peut-être en cours d'exécution).
- **Idempotence vraie** : si le contenu n'a pas changé (hash SHA-256 identique), la ligne n'est pas touchée (`updated_at` inchangé). Le résultat affiche `N unchanged`.

#### Mapping de statut

L'importeur ne pose que deux statuts de planning : `backlog` ou `done`. Les statuts d'exécution (`running`, `failed`, `current_stage`) sont gérés par le runtime et jamais produits par l'import.

| Source | Règle de mapping vers `done` | Tout le reste |
|---|---|---|
| **Markdown** | Littéral `done` uniquement (case-insensitive, trimmed) | `backlog` — y compris `in_progress`, `running`, `wip`, `closed`, vide… |
| **GitHub Projects** | Option de statut dans `done_options` (case-insensitive) — **défaut `[]`** = tout est `backlog` sauf config explicite | `backlog` — y compris `CLOSED`+`NOT_PLANNED`, "In Progress", "Blocked", OPEN… |

`done` sur une story existante n'est appliqué que si `status == 'backlog' && current_stage == null` (jamais pendant ou après une exécution). `running`/`failed` ne sont jamais produits par aucune source externe.

#### Nouveaux comportements

- **Le champ `epic` du frontmatter Markdown crée désormais des epics** : `epic: Auth` crée ou adopte un epic nommé "Auth" (l'ancien importeur ignorait ce champ).
- **Clobber de statut supprimé** : l'ancienne logique qui écrasait le statut à l'import (lignes 137-138/154-155 de `story_import.go`) est retirée. Le statut n'est plus écrasé.
- **Enrichissement in-app préservé** : `objective`, `target_files`, `acceptance_criteria` renseignés via l'UI ne sont jamais mis à null par un re-import dont la source ne porte pas ces champs.
- **`epic_id` est set-once** : une story rattachée à un epic n'est jamais déplacée vers un autre epic par un re-import.

#### Résolution des identités

| Source | Résolution à la mise à jour |
|---|---|
| Markdown | Par `(project_id, key)` — une ligne `manual` backfillée se soigne elle-même en `markdown` au premier re-import |
| GitHub Projects | Par `(project_id, source, external_id)` — le node_id opaque est stable même si le numéro d'issue change |

#### UI — import

- **Dialog sélecteur de source** (admin only) : bouton "Import planning" / "Re-import" dans le header du board ou en CTA de l'état vide.
- **Sélecteur de source** (`SelectButton`) : Markdown | GitHub Projects.
- **Onglet Markdown** : drop-zone + prévisualisation locale (parsing client-side avant tout appel API).
- **Onglet GitHub Projects** : champs `project_url` (requis), `status_field` (défaut "Status"), `done_options` (défaut vide), `epic_issue_type` (défaut "Epic").
- **Preview dry-run** : bouton "Preview" — POST `dry_run: true` — affiche un tableau par item (clé, type, action [create/update/skip/lock/fail], statut mappé, lien source, raison).
- **Import** : bouton "Import" — POST `dry_run: false` — le board se rafraîchit automatiquement.
- **Badge de provenance** : chaque carte du board affiche le badge source ; `source_url` est un lien externe vers l'item d'origine.
- **Re-import admin-gated** : `POST /planning/import` requiert le rôle admin. Sur `403`, message inline.

#### Limites connues

- **Edition amont mid-run gelée** : si une story est en cours d'exécution, le re-import ne met pas à jour son titre/spec (gelé). La modification est réappliquée automatiquement au **premier re-import après la fin du run**. Pas de badge "drift" ni de file d'attente de changements pendants.
- **`DependsOn` GitHub et ancestry multi-niveau différés** : seul le parent direct (`parent.id`) est importé en v1 ; le parcours jusqu'à l'epic ancêtre le plus proche et les dépendances natives GitHub sont différés.
- **Pas de sync planifiée** : la resynchronisation s'effectue manuellement (bouton "Re-import"). Un scheduler de sync périodique est différé.
- **`/stories/import` déprécié** : l'ancien endpoint est maintenu pour compatibilité (redirige vers le nouveau service) mais déprécié — utiliser `POST /projects/{id}/planning/import` avec `source: markdown`.
- **Concurrence** : deux imports simultanés du même projet sont last-writer-wins sur les champs cosmétiques. Le verrou row-level (`running`/`failed`/`current_stage`) protège les stories en cours d'exécution.
- **Write-back Markdown** : le write-back de statut n'est pas disponible pour les sources Markdown (format statique). Seul GitHub Projects v2 supporte le write-back.

#### Connecteur persisté et write-back de statut (GitHub Projects v2)

Un **connecteur de planning** peut être configuré par projet dans **Settings → "Tracker & sync"**. Il lie le projet à un board GitHub Projects v2 spécifique et configure le write-back de statut.

**Configuration** (API : `GET/PUT /projects/{id}/planning/connector`) :

| Champ | Rôle |
|---|---|
| `project_url` | URL du board GitHub Projects v2 |
| `status_field` | Nom du champ single-select du board (défaut "Status") |
| `done_options` | Options de statut considérées comme `done` à l'import |
| `epic_issue_type` | Type d'issue considéré comme epic (défaut "Epic") |
| `status_mapping` | Mapping interne → option GitHub : `{backlog, running, done, failed}` → option id (nullable) |
| `writeback_enabled` | Active le renvoi des transitions de statut vers le tracker |
| `post_run_comment` | Poste un commentaire avec le lien du run sur l'item tracker à chaque transition |

**Workflow utilisateur** :

1. Aller dans le projet → **Settings** → section **"Tracker & sync"**.
2. Saisir la `project_url` et le `status_field`.
3. Cliquer **"Load options"** pour sonder le board et obtenir les options réelles du champ.
4. Mapper chaque statut interne (`backlog`, `running`, `done`, `failed`) vers une option GitHub, ou utiliser **"Auto-fill"** (matching par convention, insensible à la casse).
5. Activer les toggles **"Enable write-back"** et/ou **"Post run comment"**.
6. Cliquer **"Save"** — le connecteur est persisté.

Le connecteur pré-remplit automatiquement le dialog d'import (plus besoin de ressaisir l'URL à chaque re-import).

**Write-back automatique** : à chaque transition de statut déclenchée par le pipeline (backlog → running → done/failed), si `writeback_enabled` est vrai et que l'option correspondante est mappée, le backend met à jour le champ de statut de l'issue GitHub Projects via l'API. Si `post_run_comment` est vrai, un commentaire avec le lien vers le run est posté sur l'issue.

**Contraintes** :
- Requiert une connexion GitHub (PAT) valide. Sans connexion, `PUT /planning/connector` retourne `422 PLANNING_CONNECTOR_NO_GIT_CONNECTION`.
- `writeback_enabled: true` sans aucune option mappée retourne `422 PLANNING_CONNECTOR_INVALID_MAPPING`.
- Accès réservé à l'owner du projet et aux admins globaux.

**État de synchronisation sur la story** (`writeback_status`) :

| Valeur | Signification |
|---|---|
| `disabled` | Write-back désactivé ou story sans mapping applicable |
| `pending` | Transition en file, write-back en cours |
| `synced` | Dernière transition reflétée avec succès dans le tracker |
| `failed` | Dernière tentative de write-back en erreur |

Le `WritebackStatusBadge` est affiché dans le panneau de détail de la story (visible pour `pending`, `synced`, `failed` ; masqué pour `disabled`).

**Raccourci** : le bouton **"Tracker & sync"** dans l'éditeur de pipeline renvoie directement vers la section correspondante des settings du projet.

### Connexion GitHub par PAT

Un projet peut stocker un **Personal Access Token (PAT) GitHub chiffré** dans Project Settings → "Git connection". Ce token est résolu à la place de la variable d'environnement `GITHUB_TOKEN` ou `git_token_env` pour toutes les opérations qui nécessitent un accès à GitHub : import depuis GitHub Projects v2, création de branches, PR, polling CI.

#### Coller un PAT

1. Dans le projet → **Settings** → section "Git connection".
2. Coller le token dans le champ "Personal Access Token" (masqué, toggle mask).
3. Cliquer **Save & verify** : le backend valide le token auprès de GitHub avant de le chiffrer et de le persister. Un statut `connected` avec les 4 derniers caractères du token (`…abcd`) et le type (`classic` ou `fine_grained`) s'affiche.

Le token est chiffré **AES-256-GCM** au repos (même clé que les API keys utilisateur, `ENCRYPTION_KEY`). Il n'est jamais retourné par l'API — seulement ses 4 derniers caractères sont exposés.

#### Tester la connexion

Le bouton **Test connection** re-sonde GitHub à la demande et rafraîchit le statut. Il peut être utilisé avec un token non encore sauvegardé (saisi dans le champ) ou avec le token stocké (champ vide).

#### Sens des statuts

| Statut | Signification |
|---|---|
| `unconfigured` | Aucun token stocké ; résolution se fait via `GITHUB_TOKEN` / `git_token_env`. |
| `connected` | Token validé lors du dernier "Save & verify" ou "Test connection". |
| `invalid` | Token rejeté par GitHub (401). Doit être remplacé. |
| `expired` | Token fine-grained avec `expires_at` dépassé (détecté localement). |
| `insufficient_scope` | Token valide mais manque `read:project` (et/ou `repo`/`read:org`). |

**Important — anti-déphasage** : le statut affiché est le **dernier connu** (horodatage "Last checked"). Il n'est PAS mis à jour en continu. Il se resynchronise automatiquement lors d'une vraie opération (import, branche, PR) : si GitHub retourne 401/403, le statut passe à `invalid`/`insufficient_scope` sans intervention. Utiliser "Test connection" pour forcer une vérification à la demande.

#### Scopes recommandés

| Scénario | Scopes minimum |
|---|---|
| Import GitHub Projects v2 (organisation) | `read:project` + `read:org` |
| Import + issues privées | `read:project` + `repo` |
| Branche / PR via API | `repo` |

**Fine-grained PAT recommandé** pour limiter le blast radius (portée par dépôt ou par organisation, expiry obligatoire, approuvable par l'organisation). Limitation : les fine-grained PATs ne peuvent pas lire les **Projects v2 appartenant à un compte personnel** (uniquement les projets d'organisation).

Pour les projets qui mixent les deux cas, un **PAT classique** avec `read:project` + `repo` reste accepté ; le risque (blast radius account-wide, pas d'expiry par défaut) doit être pesé.

#### Fallback d'environnement

Si aucun token n'est stocké pour le projet (ou après "Disconnect"), la plateforme se rabat sur :

1. La variable d'environnement nommée dans `git_token_env` du projet (Advanced settings).
2. La variable d'environnement `GITHUB_TOKEN` du serveur.

Le champ `git_token_env` est accessible via **Settings → Advanced — legacy env-var fallback** (masqué par défaut). Il est maintenu pour compatibilité avec les déploiements existants.

#### Qui peut connecter

Seuls l'**owner du projet** et les **admins globaux** peuvent créer, modifier ou supprimer la connexion Git d'un projet. Les autres membres voient l'état en lecture seule.

#### Déconnecter

Le bouton **Disconnect** (⚠️ derrière une confirmation) supprime le token chiffré. La résolution revient au fallback d'environnement. L'action est idempotente (no-op si aucun token n'est stocké).

#### Garde d'import

L'onglet **GitHub Projects** du dialog d'import reste désactivé tant que la connexion n'est pas `connected`. Un lien "Connect this project to GitHub first →" pointe vers les settings. Cela remplace l'ancienne erreur 422 opaque retournée lors de l'import sans token.

### Exécution de pipeline
- Lancement d'une story : substrat isolé → agent code → branche → PR
- Pipeline configurable par projet (groupes d'étapes = stages, agents, modèles, prompts)
- **Politique de transition par stage** (configurée dans l'éditeur de pipeline) :
  - **auto** : la carte avance seule au stage suivant
  - **manual** : la carte est parquée idle à l'entrée du stage, l'utilisateur la démarre via le bouton **Go** sur le board
  - **gate** : validation humaine (HITL) avant d'avancer
- Pause/reprise d'une exécution en cours
- Annulation d'un run
- Retry d'un step en échec depuis l'UI

### Retry et résilience
- Retry incrémental : l'agent reçoit le diff + l'erreur CI et corrige le code existant
- Fallback vers retry complet après 2 échecs incrémentaux
- Circuit breaker configurable (arrêt après N échecs consécutifs)
- CI polling natif via GitProvider (pas d'agent gaspillé)

### Monitoring temps réel
- Logs d'agent en streaming SSE dans le navigateur
- Suivi de la progression étape par étape (pipeline horizontale)
- Notifications Discord et webhook sur les événements

### Gates humaines (HITL)
- Pause automatique aux checkpoints configurés
- Visualisation du diff pour approbation
- Approuver ou rejeter avec raison

### Probes et halt-gate (sécurité runtime)
- **Guards** configurables par stage dans l'éditeur de pipeline : un probe (`log_silence` = heartbeat, `wallclock` = timeout, `cost_batch` = budget cumulé), un seuil, et une politique `on_fail` (halt-gate par défaut, sinon fail/retry)
- En cas de breach, le runtime **parque** la carte sur un **halt-gate** plutôt que de la laisser dériver : le run est suspendu avec une raison structurée (valeur observée vs seuil)
- Vue **Triage des halt-gates** (`/halts`) : les runs parqués sont listés et **groupés par raison de probe**, chaque carte affichant le contexte (story, stage, mesure) et les actions de résolution — **resume** (relancer le step), **override** (accepter et avancer), **skip** (passer outre), **send back** (revenir à un stage antérieur), **abort** (échouer la carte) — plus un **Resume all** par groupe

### Suivi des coûts
- Tracking des tokens et coûts par étape, run, story et agent
- Dashboard de coûts avec graphiques et agrégations
- Limites de budget configurables par projet

### Agents et runtime
- Entité Agent configurable (image, modèle, provider, prompt)
- Support multi-provider (Claude, opencode)
- Agent runtime Go avec callbacks HTTP

#### Substrat d'exécution pluggable
- L'exécution d'un agent est une couche **pluggable derrière `port.AgentRuntime`** : tous les substrats sont **égaux derrière le même port**, le substrat live est choisi par config (`SUBSTRATE`). **Docker n'est pas un cas spécial** — c'est un adapter comme les autres.
- **Docker** (`SUBSTRATE=docker`) — **défaut dev/CI**, tourne partout (pas besoin de KVM).
- **microsandbox** (`SUBSTRATE=microsandbox`) — **policy de prod** : microVM libkrun, isolation noyau (KVM) pour le code non-fiable généré par l'agent ; fidélité nested-container native (testcontainers/DinD dans l'agent). Linux/KVM-only.
- **exec** (local, sans container) et **K8s/OpenShift gVisor/Kata** (futur P4) sont ajoutables derrière le même port, **sans toucher le domaine**.

#### Image agent vs services (deux concerns distincts)
- **Image agent** = l'environnement *où* l'agent s'exécute (rootfs du microVM/container) : harness `agent-runtime` + CLI (claude/opencode) **et la toolchain** (go/node/python) pour build/test. Fournie par un **catalogue de stacks** — images ghcr digest-pinnées (go-node, node, go, python). Un microVM, comme un container, a **toujours** besoin d'une image (son rootfs).
- **Services** (db, redis, keycloak…) = **dépendances réseau**, pas le runtime de l'agent. Lancés en **sidecars** via la feature **Environment** : réseau de run isolé + conn-strings injectées dans l'env de l'agent.
- **Caveat** : sidecars-sous-microVM pas encore supportés par microsandbox (dégrade en conn-strings seules) ; le substrat **Docker** fournit les sidecars complets.

### Actions pipeline intégrées
- `agent_run` : exécution d'agent sur le substrat configuré, dispatchée via `port.AgentRuntime` (mode callback ou legacy)
- `ci_poll` : polling CI via l'API GitHub (go-github)
- `git_branch` / `git_pr` : création de branche et PR via l'**API GitHub** (go-github, token) — plus de dépendance au binaire `gh` côté backend ; l'agent commit/push avec `git`
- `hitl_gate` : gate d'approbation humaine
- `notification` : envoi de notifications
- `incremental_retry` : retry intelligent avec contexte d'erreur

### Projet de référence
- Todo app avec build, seed SQL, CI pipeline
- Baseline pour validation de la pipeline end-to-end

## Problèmes connus (non résolus)

Bugs identifiés lors d'un audit E2E mais jamais corrigés (ancienne équipe). À valider par tests avant de les traiter.

| Problème | Impact | Prio |
|----------|--------|------|
| Boucle de redirect au login | L'utilisateur ne peut pas se connecter proprement | P0 |
| Seed SQL avec hashes bcrypt cassés | Les données de dev ne fonctionnent pas | P0 |
| Endpoint /users accessible sans être admin | Faille de sécurité | P0 |
| Flow forgot/reset password cassé | L'utilisateur ne peut pas réinitialiser son mot de passe | P1 |
| Seed runs avec mauvais project UUID | Les données de démo ne s'affichent pas | P1 |
| Compteurs stories/epics incorrects sur le board | L'utilisateur voit des chiffres faux | P1 |
| Pas de sélecteur de rôle à l'édition d'un user | L'admin ne peut pas changer le rôle | P2 |
| Cost dashboard ne filtre pas par projet | Les coûts de tous les projets sont mélangés | P2 |
| Pas de dark mode | Confort visuel | P2 |
| Messages d'erreur auth cryptiques | L'utilisateur ne comprend pas ce qui échoue | P2 |

## Capacités à venir

### Budget enforcement
- Arrêt automatique de l'exécution quand le budget est dépassé

### Providers supplémentaires
- GitLab, Bitbucket, Azure DevOps (Git providers)

### UX avancée
- Dashboard principal avec KPIs (taux de succès, coûts, activité)
- Autonomie progressive (auto-approve basé sur le track record)

### CLI
- Interface ligne de commande : run, status, logs, approve/reject

### Notifications avancées
- Email (SMTP)
- Canaux configurables par type d'événement
