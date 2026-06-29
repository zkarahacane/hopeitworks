# ADR — Connexion Git par token PAT chiffré

**Statut :** Accepté  
**Date :** 2026-06-27  
**Contexte :** feat/git-connection (P0–P2 mergés sur feat/git-connection ; P3 e2e+docs = cette PR)

---

## Problème

La plateforme a besoin d'accéder à GitHub (import depuis GitHub Projects v2, `git_branch`, `git_pr`, `ci_poll`) pour chaque projet. Avant cette décision, le token était une variable d'environnement globale (`GITHUB_TOKEN`) ou une variable nommée par projet (`git_token_env`). Cette approche :

- force un token partagé entre tous les projets sur l'instance ;
- empêche chaque projet d'avoir son propre compte ou organisation GitHub ;
- ne permet pas de renouveler un token spécifique sans redéploiement ;
- masque les erreurs de token (les opérations échouent avec une 422 opaque).

---

## Décision

### D1 — PAT par projet, chiffré au repos (méthode token)

Un **Personal Access Token (PAT)** GitHub est stocké par projet dans la table `git_connections` sous forme de blob `nonce+ciphertext+tag` (AES-256-GCM via `pkg/crypto`), la même clé (`ENCRYPTION_KEY`) que les API keys utilisateur et les credentials existants. Aucune nouvelle clé, aucune nouvelle infra crypto.

**GitHub App rejeté pour v1.** Rationale : l'App nécessite l'enregistrement d'une App opérateur, la persistance de la clé privée RS256, le mint de JWT, les webhooks `installation.*` avec HMAC, un callback OAuth avec nonce CSRF, et le mint de tokens d'installation 1h mis en cache. Cela représente ~15 nouveaux fichiers et retarde toute connexion en app. Le PAT donne 80 % de la valeur (token par projet, géré dans l'UI) sans ce coût. Le seam (D2 ci-dessous) est conçu pour accueillir l'App en drop-in futur.

### D2 — Un seam unique : `GitCredentialResolver`

Les deux factories qui avaient leurs propres fonctions `resolveGitToken`/`resolveGitHubToken` (duplication validée dans `git/provider_factory.go:54` et `planning/factory.go:62`) sont remplacées par un **port injecté** :

```go
type GitCredentialResolver interface {
    TokenForProject(ctx context.Context, projectID uuid.UUID) (GitToken, error)
}
```

`GitConnectionService` implémente ce port. La résolution est :
1. Ligne `git_connections` (KIND=`pat`) → déchiffrement AES-256-GCM → `GitToken.Value`.
2. Fallback : `os.Getenv(project.GitTokenEnv)` → `os.Getenv("GITHUB_TOKEN")`.

Les factories ne connaissent que `GitCredentialResolver` — le KIND (`pat`, ou un futur `github_app`) est opaque au-dessus du seam.

### D3 — Table KIND-discriminée (`git_connections`)

```sql
CREATE TABLE git_connections (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider    VARCHAR(16) NOT NULL DEFAULT 'github'
                  CHECK (provider IN ('github','gitea','gitlab','bitbucket')),
    kind        VARCHAR(16) NOT NULL CHECK (kind IN ('pat')),  -- widened by App migration
    encrypted_secret BYTEA NULL,   -- nonce+ciphertext+tag (AES-256-GCM)
    secret_last4     VARCHAR(8) NULL,
    token_type       VARCHAR(16) NULL,  -- classic | fine_grained | unknown
    scopes           TEXT[] NOT NULL DEFAULT '{}',
    status           VARCHAR(24) NOT NULL DEFAULT 'unconfigured',
    account_login    VARCHAR(255) NULL,
    expires_at       TIMESTAMPTZ NULL,
    last_validated_at TIMESTAMPTZ NULL,
    validation_error  TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT git_connections_uq_project UNIQUE (project_id)
);
```

La colonne `kind` agit comme discriminant. Une migration additive future (e.g. `000044`) élargit `CHECK (kind IN ('pat','github_app'))` et ajoute les colonnes App (`installation_id`, etc.) **sans réécrire la table ni le seam**. Un projet sans ligne se comporte exactement comme avant (fallback env).

### D4 — Statut advisory + revalidation live (anti-déphasage)

Le statut stocké en base est **dernier-connu**, pas ground truth. GitHub est la source de vérité, consultée :
- **À la demande** : via "Test connection" (POST `.../test`) — sonde GitHub, met à jour `last_validated_at` et `status`.
- **À l'usage** : lors d'une vraie opération (`git_branch`, `git_pr`, `ci_poll`, import), une réponse 401/403 retourne en `invalid`/`insufficient_scope` via `SetGitConnectionValidation`. Transient (429/5xx) : pas de changement de statut.
- **Lazy expiry** : si `expires_at < now()`, le statut résout à `expired` sans appel réseau.

L'UI n'affiche jamais un "connected" nu — toujours accompagné de `last_validated_at` ("Last checked …"). Un polling actif n'est pas nécessaire grâce à l'auto-correction à l'usage.

### D5 — Validate-before-store

Sur PUT, si `validate: true` (défaut), le backend sonde GitHub **avant** de chiffrer et persister. 401/403 définitif → `422`, pas de persistance. Transient (5xx/429) → `503`, le token existant éventuellement valide n'est pas écrasé.

### D6 — Pinning du host probe (sécurité)

Le validateur ne transmet le token qu'à un host **de confiance** :
- GitHub API : `GITHUB_API_BASE_URL` (défaut `https://api.github.com` ; override pour GitHub Enterprise Server).
- Gitea : URL dérivée de `project.repo_url` — accessible uniquement aux admins, qui ont aussi défini l'URL.

Le token ne sort jamais vers une URL arbitraire fournie par un utilisateur.

### D7 — Autorisation : owner du projet ou admin global

Chaque handler git-connection vérifie `IsAdmin(ctx) || project.OwnerID == actorUserID`. Plus strict que la simple appartenance au projet : connecter un client GitHub à un projet est une action de configuration privilégiée. Suit le même pattern que la suppression de projet.

---

## Alternatives rejetées

### GitHub App en v1

L'App est la bonne réponse multi-tenant (blast radius minimal, révocation par déinstallation, tokens 1h éphémères, auditabilité). Elle est documentée comme **cible future** et tombera en drop-in derrière le même seam (D2) et la même table (D3 avec migration additive). Rejetée pour v1 uniquement pour le coût de mise en place (voir D1).

### Variable d'env nommée par projet uniquement (`git_token_env` comme seul mécanisme)

Maintenu comme fallback de compatibilité (D2 §3.4). Rejeté comme mécanisme primaire : pas d'UI de gestion, pas de validation au save, pas de rotation sans redéploiement.

### Colonne `env_var` KIND dans `git_connections`

YAGNI : le fallback env est dans le code du resolver, pas dans une ligne de table. Une ligne `env_var` ajoute de la complexité sans valeur (la variable d'env est déjà lisible via `project.GitTokenEnv`).

---

## Gaps documentés (non bloquants pour v1)

| Gap | Mitigation v1 | Plan futur |
|---|---|---|
| Clé unique `ENCRYPTION_KEY`, pas de versioning | Rotate = re-entrer les tokens (documenté). Fail-fast sur clé vide lors d'un PUT (B1). | Tâche cross-cutting : `keyID`-versioned blobs, s'applique aussi aux `credentials` et `user_api_keys`. |
| PAT classique : blast radius account-wide | Docs + UI recommandent fine-grained PAT. | GitHub App en drop-in (D1). |
| Pas de circuit-breaker run sur 401/403 | Status `invalid` set, l'opération retourne une erreur (le run échoue). | Halt-gate automatique sur 401/403 pendant le run (follow-up). |
| Rotation automatique | Manuelle. | GitHub App tokens (1h, révocables). |

---

## Conséquences

- `git/provider_factory.go` et `planning/factory.go` n'ont plus de token-resolution inline (fonctions `resolveGitToken`/`resolveGitHubToken` supprimées).
- Un projet sans ligne `git_connections` est byte-for-byte identique à l'état avant cette feature (zero migration risk).
- Le seam `GitCredentialResolver` est le **seul point d'entrée** vers un token git pour toute la plateforme — le token ne traverse jamais la couche handler ni les containers d'agents.
- L'App GitHub peut être livrée par une migration additive + une implémentation de `GitCredentialResolver` sans toucher le domaine ni les handlers existants.
