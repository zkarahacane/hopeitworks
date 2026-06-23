# hopeitworks — Vision Produit

*Maintenu par François. Dernière mise à jour : 2026-06-23.*

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
                         Agent (container Docker isolé)
```

- **Project** : un repo Git avec sa configuration (pipeline, provider, budget)
- **Epic** : un lot de stories liées, avec un graphe de dépendances (DAG)
- **Story** : une user story avec des critères d'acceptation testables
- **Run** : une exécution complète de la pipeline pour une story
- **Step** : une étape de la pipeline (code, CI, review, merge...)
- **Agent** : un agent IA qui exécute une étape dans un container isolé

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
- Import de stories depuis des fichiers markdown
- Éditeur de stories (création et édition dans l'UI)
- Epics avec calcul de DAG et visualisation des dépendances
- Lancement d'un epic entier avec exécution parallèle

### Exécution de pipeline
- Lancement d'une story : container Docker → agent code → branche → PR
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
- Entité Agent configurable (image Docker, modèle, provider, prompt)
- Support multi-provider (Claude, opencode)
- Agent runtime Go avec callbacks HTTP
- Images Docker multi-stack (go-node, node, go, python)

### Actions pipeline intégrées
- `agent_run` : exécution d'agent en container (mode callback ou legacy)
- `ci_poll` : polling CI (GitHub Actions, GitLab)
- `git_branch` / `git_pr` : création de branche et PR
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
