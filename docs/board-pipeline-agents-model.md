# Board ↔ Pipeline ↔ Agents — Modèle de référence

> **But du doc.** Sortir du flou la relation entre le *board/Kanban*, le *pipeline* et les *agents*. C'est le modèle conceptuel qui sous-tend l'exécution — distinct du doc runtime (`agent-runtime-capabilities-plan.md`, qui traite *comment* un agent s'exécute dans son sandbox). Ici on traite *quel est l'état durable*, *comment il bouge*, et *comment l'humain reste maître*.
>
> **Statut.** Modèle convergé après discussion + passe critique adverse (regard delivery non-technique). Pas encore implémenté. Les références code décrivent l'existant (point de départ), pas une cible figée.

---

## 1. TL;DR — le modèle en une image

```
Stage (durable, = source de vérité)
  ├─ colonne Kanban
  ├─ segment de pipeline   (les steps automatisés qui tournent quand la carte y entre)
  ├─ politique de transition   auto │ manual │ gate(HITL)
  └─ guards / probes           heartbeat │ timeout │ cost │ blast-radius │ …
                                    └─ on_fail ▸ halt-gate (par défaut) │ fail │ retry

Story  = 1 carte du board.  Epic = batch de stories (DAG via depends_on).
Lifecycle (Backlog/Running/Done/Failed) = coarsening DÉRIVÉ des stages, pas un objet séparé.
Tracker externe (Jira/GH/GL/markdown) → mapping DÉCLARATIF statut↔stage ; un master par source.
Arbre live + Kanban = PROJECTIONS reconstruites depuis Postgres. Jamais la vérité.
```

Une phrase : **le Stage est l'unité durable de vérité ; tout le reste (colonnes, exécution, sync tracker, arbre live, gates de sécurité) en découle.**

---

## 2. Décision fondatrice : Stage = source de vérité

Aujourd'hui deux concepts sont **découplés**, et c'est ce qui crée le flou :

- le **pipeline** : suite d'actions techniques (`git_branch → agent_run → git_pr → ci_poll → hitl_gate`) — défini en YAML, un seul par projet (`pipeline_configs`, cf. `backend/internal/domain/model/pipeline_config.go`, migration `000007`) ;
- les **colonnes du board** : un lifecycle macro dérivé de `story.status ∈ {backlog, running, done, failed}` (cf. `backend/internal/domain/model/story.go`), re-projeté en 5 colonnes côté front par `boardColumn()` (`frontend/src/stores/stories.ts`).

Le pont actuel = « l'étape courante affichée *dans* la carte ». On **fusionne** les deux concepts via un seul objet : le **Stage**.

> **Invariant.** La vérité vit en **Postgres** (définition des stages, `current_stage` par story, runs/steps, log append-only `events`). Le **container agent est jetable**, le **pod orchestrateur est jetable**, le **navigateur est jetable**. Le Kanban et l'arbre live ne sont que des **projections** reconstruites depuis Postgres.

Pourquoi non-négociable : si quelqu'un supprime le pod/container, on ne doit **rien** perdre. L'état d'avancement est en base, pas dans un process vivant.

---

## 3. L'objet Stage (3-en-1 + guards)

Un Stage est simultanément :

1. **une colonne** du board ;
2. **un segment de pipeline** — les steps automatisés exécutés pendant que la carte est dans ce stage ;
3. **une politique de transition** : comment la carte *quitte* le stage.

```
Stage {
  id, name                         # human-meaningful (≠ "git_branch")
  segment:    [ step, step, … ]    # 1 group actuel ≈ 1 stage (voir §11)
  transition: auto | manual | gate
  guards:     [ … ]                # §6
}
```

### Politique de transition

| Politique | Comportement | Sert… |
|---|---|---|
| `auto` | tourne et avance seule | l'overnight non-supervisé |
| `manual` | la carte **entre** dans le stage et **attend idle** un clic humain « Go » | board collaboratif piloté humain |
| `gate` | le travail finit, mais un humain doit **approuver** avant que la carte sorte (HITL) | gouvernance / review |

> **Décision UX.** Sur `manual`, la carte **entre dans le stage et attend idle** (tu la *vois* « In Dev, pas démarrée »), elle n'attend pas dans le stage précédent. Intuition Kanban : la carte est dans la colonne, le boulot pas commencé.

### Le bouton « Go » / « Développer » : deux points d'action distincts

| | Quoi | Visible quand |
|---|---|---|
| **Entry trigger** (Go) | démarre le segment d'un stage `manual` | carte en Backlog (→ lance dans son stage d'entrée), ou posée idle dans un stage `manual` |
| **Exit gate** (Approve/Reject) | autorise la carte à *quitter* | segment terminé, transition = `gate` |

Règles :
- **Cliquable** sur une carte non-running (Backlog, ou idle dans un stage manuel).
- **Pas cliquable** pendant qu'un segment tourne (→ devient Pause/Cancel) ni sur un gate (→ Approve/Reject).
- **Un clic peut porter loin** : Go lance le stage courant *puis enchaîne tous les stages `auto`* jusqu'au prochain `manual`/`gate`. Tout `auto` → un clic → Done. Un gate présent → un clic → la carte se gare au gate.
- La dispo du bouton se **dérive de `current_stage` + politique** (donc de Postgres) → refresh navigateur / reboot pod la recalcule correctement.

---

## 4. Lifecycle = vue dérivée

On ne choisit **pas** entre « colonnes = lifecycle » et « colonnes = étapes ». On **dérive** le lifecycle des stages :

- `backlog` = avant le 1er stage ;
- `running` = dans un stage non-terminal ;
- `done` = après le dernier stage ;
- `failed` = erreur non récupérée.

Le board peut afficher **l'une OU l'autre granularité** sans modèle de données séparé : vue **macro** (lifecycle) pour le dashboard « réveil », vue **détaillée** (stages) pour le travail d'équipe. Même donnée.

---

## 5. Entry by readiness

**Où une carte entre dans le pipeline dépend de sa maturité.**

- Story richement spécifiée (AC complètes, style BMAD `ready-for-dev`) → entre directement en **`In Dev`**.
- Issue GitHub one-liner → entre en **`Needs Spec`**, où un agent de cadrage la prépare *avant* le dev.

Ça unifie l'« enrichment » du doc connectors (qui était posé comme un mur bloquant) : ce n'est plus un gate, c'est **le stage d'entrée selon la richesse**. L'échelle de richesse = « à quel stage tu rentres ».

> **Insight stratégique** (issu de la critique delivery) : *le goulot n'est presque jamais l'exécution* — c'est les requirements flous. `Needs Spec` / entry-by-readiness est donc la **valeur la plus défendable** du produit, pas la vitesse brute. À mettre en avant.

---

## 6. Guards / Probes — la couche de sécurité

Un agent LLM ne *crashe* pas proprement : il **stalle, loope, ou brûle du budget en silence**. Les probes sont la réponse *ingénierie* au reproche #1 de la critique (« overnight autonome + plausible-but-wrong = pire combo en delivery »). Vocabulaire K8s, transposé à l'agent-dans-le-stage.

### Famille 1 — opérationnelles (cheap, prioritaires — c'est ça qui rend l'overnight sûr)

| Probe | Détecte | Note |
|---|---|---|
| **heartbeat / liveness** | agent stuck (pas de battement N sec) | **la plus importante pour de l'IA** ; ✅ **buildable maintenant** — dérivé du flux de logs incrémental existant (« pas de `log.emitted` depuis N sec » ; `events.created_at` Postgres). Le flux de logs EST le heartbeat. Un ping dédié = amélioration future runtime, pas requis |
| **wall-clock timeout** | step/stage qui ne finit pas | le « timeout » demandé ; **board-side** (timer sur `step.started`) → ✅ **buildable** |
| **cost ceiling** | runaway token/coût | coût émis aujourd'hui mais **terminal-only** (1× en fin de step) → ✅ ceiling **batch/post-step** buildable (halte le reste de l'epic) ; ⚠️ circuit-breaker **mid-run** = besoin d'émission coût incrémentale (petit ajout runtime, pas un blocage de fond) |

### Famille 2 — sémantiques (smart, à phaser — anti « plausible-but-wrong »)

| Probe | Détecte |
|---|---|
| **blast-radius** | fichiers touchés hors `target_files` / `scope` |
| **loop detection** | mêmes tool-calls répétés N fois |
| **diff sanity** | diff énorme, zéro test… (souvent via *LLM judge* = 2e agent cheap → coût/latence) |

### Slotting dans le modèle

Une probe est un **4ᵉ déclencheur de transition**, à côté de `auto/manual/gate`. Action `on_fail` :

```
guards: [
  { heartbeat,    timeout: 120s,         on_fail: halt-gate }
  { wallclock,    max: 30m,              on_fail: fail }
  { cost,         max_usd: 5,            on_fail: halt-gate }
  { blast_radius, scope: target_files,   on_fail: halt-gate }
]
```

- **`halt-gate` = défaut.** Pause le run **au stage durable courant** + lève un HITL. Au réveil : *« S-03 stalled @ In Dev : pas de heartbeat 2 min — intervenir ? »*. Carte **parquée avec une raison**, pas un mess. En overnight, *parquer-avec-raison > tuer* (beaucoup de stalls sont récupérables d'un clic).
- `fail` = échec sec + fail-fast la couche epic si des deps en dépendent.
- `retry` = retry borné (`retry_count` / `retry_type` existent déjà sur `run_steps`).

Un halt = **un event durable** → le pod peut mourir, tu te réveilles quand même devant la carte parquée.

---

## 7. Halt-gate vs review-gate, et le déblocage humain

**Débloquer une halt-gate ≠ approuver une review-gate.**

| | Review gate | Halt-gate (probe) |
|---|---|---|
| État du travail | **terminé**, en attente d'aval | **interrompu** en plein vol |
| Question | « c'est bon ? » | « comment on récupère ? » |
| Actions | Approve / Reject | jeu enrichi ↓ |

### Jeu d'actions de déblocage (halt-gate)

- **Resume / Retry** — relance le step ; fresh, ou **avec bornes ajustées selon la raison** (cost ceiling → « retry avec +budget »).
- **Override & continue** — accepte le partiel, avance ; pour le faux positif (agent fini mais pas signalé). **Explicite + audité**, jamais silencieux.
- **Take over** — l'humain reprend à la main (ouvre le workspace, finit, marque done). *L'accès workspace = runtime ; on définit juste l'action.*
- **Send back** — renvoie à un stage amont (ex. `Needs Spec`) si la raison révèle une spec foireuse (un halt blast-radius = `scope` faux).
- **Skip stage** — saute ce stage si finalement inutile.
- **Abort** — tue la carte, fail-fast la couche epic si besoin.

Chaque action = **transition de stage durable** + enregistre `resolved_by` (humain nommé) + la raison.

### UX du réveil

1. **La raison pré-suggère le remède** : cost → *resume +budget* ; heartbeat → *retry fresh* ; blast-radius → *review diff / fix scope* (pas de retry aveugle).
2. **Triage en batch** : N cartes parquées, groupées **par raison**, actions de masse (« 3 ont tapé le plafond → bump + resume all »). Sert le persona consultant multi-clients.
3. **Auto-recovery borné** (phase 2, opt-in) : pré-autoriser des récup pour qu'elles ne deviennent pas des gates (« heartbeat perdu → retry fresh 1× avant de lever le gate »). **Défaut conservateur** : aucune auto-recovery.

### Réutilisation de l'existant

Le HITL existe déjà : `hitl_requests` (`gate_type`, `status`, `resolved_by`, `rejection_reason`, `diff_content` — migration `000020`, `model/hitl.go`). La halt-gate = un **variant** (`gate_type: probe_halt`) avec un jeu de résolution plus large qu'approve/reject. Gate en base → débloquable **quand tu veux** (« batch-resolve au café »).

---

## 8. Durabilité & résilience

- **Vérité = Postgres** : stages, `current_stage`, runs/steps, `events` append-only.
- **Container agent meurt** en plein step → step marqué interrompu, **reprenable depuis le DB** (l'executor a déjà « skip if completed » + relecture du pause-state à chaque itération — `service/pipeline_executor.go`). On ne repart pas de zéro.
- **Pod orchestrateur meurt** → jobs River durables (en Postgres) repris au boot. Lance l'epic, le pod reboote à 3h, tu te réveilles quand même avec un résultat.
- **`events` append-only** = journal rejouable (migration `000006` + trigger NOTIFY → SSE `sse_handler.go`). Refresh / nouveau client / rebuild → tout se reconstruit. Event-sourcing light.
- **Stop propre du runtime** rend les probes *actionnables* : kill via le `Stop()` du runtime (hyperviseur sur substrat KVM/microsandbox, **sinon** isolation adapter gVisor/K8s — substrat-dépendant, ne pas sur-promettre « microVM »). **Contrat requis : stop gracieux, sans écriture partielle corrompue**, pour que le halt-gate laisse un run proprement reprenable (les clones CoW par-run du runtime limitent déjà le blast radius d'un stop sale). Limites CPU/mem/IO enforced (le budget-probe a le temps d'agir), isolation FS (le blast-radius a un sens, dégâts contenus).

> **Stop runtime propre + probes + stages durables = le trio qui rend l'exécution nocturne réellement sûre.**

---

## 9. Frontière avec le runtime (l'autre doc — ne pas empiéter)

| | Possède |
|---|---|
| **Runtime** (`agent-runtime-capabilities-plan.md`) | *comment* l'agent s'exécute dans le sandbox/microVM, l'enforcement des limites, l'**émission** des signaux (heartbeat, cost, exit, events ressources) via le protocole callback, l'exécution du kill, l'accès workspace |
| **Board/Pipeline** (ce doc) | *l'état durable* (stages, runs), la **définition** des probes/policies (seuils + actions), la **réaction** (→ transition de stage durable), le Kanban, les gates, le mapping tracker |

Contrat : le runtime **émet** sur son canal callback → un **ingesteur côté board traduit callback → `events`** → l'évaluateur de policy **réagit**. (Le runtime possède le mot « callback », le board possède « events » ; seul le board nomme le pont — à expliciter pour qu'aucun côté ne suppose que l'autre fait la traduction.) Invariant conservé : **domaine = policy (data)**, exécution de probe/infra = runtime/adapter. Jamais d'infra (DinD/microVM) dans le domaine.

> **Couture — dépendances à lever côté runtime** (passe de cohérence des 2 docs, juin 2026) :
> - **G1 — émission de signaux (nuancé après vérif du CODE, pas que le doc).** Le **code actuel émet déjà** via callbacks : `logs` **incrémental** (→ `events.log.emitted` + `log_tail`, timestampé), `cost` **terminal-only** (→ `cost_records`), `status` terminal. Donc **liveness (gap de logs), wallclock et cost-ceiling batch sont buildables maintenant** — pas bloqués. Le gap est sur le *plan* runtime : le port `AgentRuntime` n'expose que `Launch/Wait/Stop/Provision/SupportedCapabilities` (aucune surface d'émission formalisée) → **risque de régression** à la réécriture + manques à porter au plan : **cost incrémental** (pour le circuit-breaker mid-run), **heartbeat dédié**, **resource-pressure**. C'est un *don't-regress + extend*, pas un blocage du socle.
> - **G2 — sémantique de `Stop()`.** Le port expose `Stop()` mais sans contrat « gracieux / non-corrompant ». Le halt-gate l'exige pour un resume propre. À pin côté runtime.
> - **Pas de contradiction** sur l'invariant durabilité ni la frontière domaine/adapter — les deux docs sont alignés.

---

## 10. Intégration trackers externes : mapping déclaratif, **zéro code utilisateur**

Ligne de partage :

> **Code pour *parser*, data pour *mapper*.**
> - **Parsing** (Jira ADF → objective, GH body → AC…) = code adapter, écrit/maintenu **par nous**, par source. Pas exposé.
> - **Mapping statut↔stage** = **config déclarative**, fixée **par l'utilisateur** par connexion : table `{statut externe ↔ stage interne}`, validée contre un schéma. **De la donnée, pas de l'exécutable** (refus du JS custom = surface de faille).

### Master & write-back (ajustement post-critique)

Le reproche le plus fort de la critique : *deux sources de vérité qui « se synchronisent » = deux sources de vérité* → ton board dit « done », le Jira client dit « in progress » en SteerCo.

> **Un master par source, choisi explicitement.**
> - Tracker connecté → *il* est master du **planning** ; nos stages drivent l'**exécution**.
> - **Write-back = action explicite, auditable, qui crie quand elle échoue** — jamais une sync auto silencieuse. Différé (v3 du doc connectors).
> - Pas de tracker (markdown / in-app) → *on* est master.
> - On ne promet **jamais** de « sync bidirectionnelle transparente ».

Le **Stage interne est la lingua franca** entre tracker externe et moteur d'exécution.

---

## 11. Le delta code (ce n'est pas un rewrite)

Les os sont là. Cartographie existant → cible :

| Concept cible | Existant | Delta |
|---|---|---|
| **Stage** | `pipeline_configs` a déjà des `groups` (steps parallèles intra-story) quasi inutilisés (`model/pipeline_config.go`) | élever `group → stage` : nom human-meaningful + politique de transition + guards. **1 group ≈ 1 stage ≈ 1 colonne.** |
| **`current_stage` sur la story** | `story.status` (4 valeurs) | ajouter `current_stage` ; `status` devient **dérivé** |
| **Transitions** | executor déroule les steps en ordre, suspend déjà sur HITL (`pipeline_executor.go`) | émettre `stage.entered` au passage de frontière de group |
| **Guards/probes** | circuit breaker projet-level + retry sur steps | évaluateur de policy consommant les signaux runtime (heartbeat/cost) via `events` |
| **Halt-gate** | `hitl_requests` (approve/reject) | variant `probe_halt` + jeu de résolution enrichi ; tous enregistrent `resolved_by` |
| **Board colonnes** | `boardColumn()` hardcodé lifecycle (`stores/stories.ts`) | rendre les colonnes depuis les stages du pipeline + toggle macro/détail |
| **Éditeur de stages** | `PipelineConfigView.vue` édite déjà groups→steps | enrichir : nom de stage, politique, guards. **Templates + éditable** (cf. §13) |
| **Arbre live** | `EpicDagView.vue` (VueFlow) + `runtimeStream.ts` (SSE) | reste une projection ; ajoute le rendu des halts/guards |

---

## 12. La critique delivery : ce qu'on plie / ce qu'on parke

Passe critique adverse (regard directeur delivery **non-technique**). Verdict : *« outil d'exécution qui cosplay une plateforme de delivery ; sa feature phare était son plus gros risque »* — modèle sorti **renforcé**.

**Plié dans le modèle :**
- **« Wake up, it's done » → « wake up, voilà un batch relisable + coût + ce qui a passé / bloqué ».** « Done » = verbe humain. Policy par défaut : tout ce qui produit du shippable s'arrête sur un **gate review avant merge** ; auto-merge = opt-in conscient.
- **One master par source / write-back explicite** (§10).
- **Accountability + audit** (son #1 « sinon inadoptable en client ») : les os existent (`events` immuable, `resolved_by` nommé). Manque, cheap : **owner nommé par carte** (champ assignee **absent** aujourd'hui), chaque approbation enregistre *qui*, events **exportables** (forensic).

**Parké explicitement (phase adoption entreprise, ≠ v1-solo) :**
- reporting/burn-up stakeholders, explication au client, SteerCo, dashboards de capacité/WIP, DoD « exit-by-quality » formel.
- *Pourquoi parké :* le user v1 = consultant tech solo qui merge ses propres PRs (la critique elle-même : « power-tool for one technical person »). Ne pas laisser la gouvernance entreprise bouffer le v1.

**Désaccords assumés avec la critique :**
- « Pick one master » absolu → nuancé en « un master par source + write-back explicite non-silencieux ».
- « Story pas atomique » (splits/sous-tâches mid-flight) → vrai mais couvert au gros par `depends_on`/Epic ; split en cours de route = **plus tard**, pas un bloqueur d'adoption.

---

## 13. La v1 — plus petit pas crédible

Issu du chemin d'adoption de la critique (et de la ligne « construire direct, livrer fin ») :

> **Un seul stage, gate final, l'humain merge, aucune sync tracker.**

Pointer un repo → ramasser **une** story bien spécifiée → l'agent implémente → ouvre une PR → **stop**. L'humain review et merge comme aujourd'hui. Prouve une bonne PR, avec un **nom sur l'approbation**. C'est *littéralement* la plus petite version du modèle Stage (1 stage → gate → humain merge).

Ce qui **tue l'adoption day-one** (à éviter absolument) : la sync qui écrase un statut Jira ; un agent qui merge sans supervision ; un échec inexplicable sans trace ; le board qui diverge de celui du client.

**Stages d'entrée du modèle** : commencer par **templates prêts** (« Dev simple », « Dev + Review + QA », « BMAD ») **éditables** ensuite → zéro-config pour démarrer, libre après.

---

## 14. Décisions actées (récap)

1. **Stage = source de vérité** ; lifecycle dérivé. Vérité en Postgres, container/pod jetables.
2. **Un pipeline par projet** pour démarrer, derrière un **port** (résoudre « pour cette story, quel pipeline + quel stage d'entrée ») pour ouvrir par-type/par-source plus tard sans toucher l'executor.
3. **Epic/Story** = primitive (plus petit dénominateur commun de tous les trackers), dé-BMAD-isée à l'UI.
4. **Politique de transition** `auto/manual/gate` = le knob qui sert overnight ET équipe.
5. **Carte entre-et-attend-idle** sur stage `manual`.
6. **Guards/probes** = 4ᵉ déclencheur ; **`halt-gate` par défaut** sur probe-fail.
7. **Mapping tracker déclaratif** ; **un master par source** ; write-back explicite & différé.
8. **Overnight = batch relisable**, jamais « done » aveugle ; gate review avant merge par défaut.

## 15. Questions ouvertes / à trancher plus tard

- Per-story / per-type pipelines (le port le permet ; pas v1).
- Split d'une story en cours d'exécution.
- Auto-recovery borné (phase 2) : quelles récup pré-autorisables, quels défauts.
- Profondeur des probes sémantiques (LLM judge : coût vs valeur).
- Modèle d'owner/assignee (par carte ? par stage ? par approbation ?).
- Gouvernance entreprise (reporting, WIP, capacité) : à reprendre à la phase adoption équipe.
