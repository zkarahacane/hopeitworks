# GitHub Project Board

Private board: https://github.com/users/zkarahacane/projects/1

## Status Pipeline (colonnes Kanban)

```
Backlog → Specified → Architected → In Progress → Review → Testing → Done
```

| Status | Signification |
|--------|---------------|
| Backlog | Pas encore spécifié |
| Specified | US fonctionnelle écrite par François |
| Architected | Specs techniques produites par architectes |
| In Progress | Devs implémentent |
| Review | Code review en cours |
| Testing | Tests E2E / démo en cours |
| Done | Mergé et validé |

## Labels agent

Chaque agent ajoute son label quand il a terminé son travail sur une issue :

| Label | Agent | Quand |
|-------|-------|-------|
| `agent:francois` | François | US fonctionnelle écrite |
| `agent:arch-back` | Architect backend | Specs techniques backend produites |
| `agent:arch-front` | Architect frontend | Specs techniques frontend produites |
| `agent:dev-back` | Dev backend | Code implémenté |
| `agent:dev-front` | Dev frontend | Code implémenté |
| `agent:code-review` | Code review | Review passée |
| `agent:test-demo` | Test/Demo | Tests E2E / démo validée |

## Labels domaine et priorité

- `domain:back`, `domain:front`, `domain:shared`, `domain:infra`
- `P0` (critical), `P1` (important), `P2` (nice to have)

## Champs Project

| Champ | Type | Valeurs |
|-------|------|---------|
| Status | Single select | voir pipeline ci-dessus |
| Epic | Single select | noms d'epics |
| Domain | Single select | back, front, shared, infra |
| Priority | Single select | P0, P1, P2 |

## Requêtes utiles (gh CLI)

```bash
# Toutes les issues du projet
gh issue list --project "hopeitworks Board"

# Stories spécifiées mais pas encore architected
gh issue list --label "agent:francois" --label "domain:back" --no-label "agent:arch-back"

# Stories implémentées mais pas reviewées
gh issue list --label "agent:dev-back" --no-label "agent:code-review"

# Stories bloquées (backlog, P0)
gh issue list --label "P0" --state open

# Ajouter un label agent après avoir fini son travail
gh issue edit <number> --add-label "agent:francois"

# Changer le status dans le Project
gh project item-edit --project-id PVT_kwHOAgh3-84BQaMD --id <item-id> --field-id PVTSSF_lAHOAgh3-84BQaMDzg-iZZI --single-select-option-id <option-id>
```

## Status option IDs (pour les mises à jour programmatiques)

| Status | Option ID |
|--------|-----------|
| Backlog | `0ba7d610` |
| Specified | `5e465c31` |
| Architected | `e24039db` |
| In Progress | `2e39b2c2` |
| Review | `7b99c4ec` |
| Testing | `f0f8ec76` |
| Done | `2fce4fa9` |

Project ID: `PVT_kwHOAgh3-84BQaMD`
Status field ID: `PVTSSF_lAHOAgh3-84BQaMDzg-iZZI`

## Workflow type pour créer une issue et l'ajouter au board

```bash
# 1. Créer l'issue
gh issue create --title "feat: description" --label "domain:back" --label "P1"

# 2. L'ajouter au project (récupère l'item ID)
gh project item-add 1 --owner zkarahacane --url <issue-url>

# 3. Setter le status
gh project item-edit --project-id PVT_kwHOAgh3-84BQaMD --id <item-id> \
  --field-id PVTSSF_lAHOAgh3-84BQaMDzg-iZZI --single-select-option-id 0ba7d610
```
