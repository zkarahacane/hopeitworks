# Plan P3 — Adapter de substrat microsandbox (microVM)

> Sous-phase finale de la refonte runtime (`docs/agent-runtime-capabilities-plan.md`, décision actée #14 : 1er adapter de substrat = **microsandbox** / libkrun microVM). Établi par un workflow de design read-only (recherche API microsandbox + cartographie codebase) le 2026-06-23.
>
> **État** : P3a (squelette) **mergé**. P3b (appels microVM réels via le SDK Go, derrière `//go:build microsandbox`) **implémenté** sur `feat/p3b-microsandbox` — build par défaut inchangé (fallback `ErrNotBuilt`), SDK `github.com/superradcompany/microsandbox/sdk/go v0.5.9` pinné, Dockerfile.microsandbox + override compose `/dev/kvm`, harness de validation `cmd/microsandbox-smoke` + test taggé. Le build taggé exige un host KVM/HVF avec libkrun (cgo) — non exerçable en CI. P3c (bascule du flow live `agent_run`→`AgentRuntime`) reste à faire.

---

## Pourquoi microsandbox

microVM (libkrun) → **vrai kernel** : testcontainers/DinD natifs, perf quasi-native, isolation plus forte que les containers partagés. C'est le défaut visé quand KVM est disponible (host Docker actuel, bare-metal, OpenShift Virt). L'adapter K8s+gVisor (sans KVM) reste un fallback ultérieur (P4 du plan directeur). Invariant conservé : tout reste **K8s-Pod-exprimable** (sidecars-in-Pod, jamais DinD comme hypothèse de domaine).

## Surface des ports (état réel sur `develop`)

Deux ports coexistent — le scaffolding cible le bon :

- **`port.AgentRuntime`** (`backend/internal/domain/port/agent_runtime.go`) = port **CIBLE**, agnostique, **non branché** sur le flow live. 5 méthodes :
  - `Provision(ctx, model.CapabilitySpec) (model.ProvisionResult, error)`
  - `Launch(ctx, port.RunSpec) (port.RunHandle, error)`
  - `Wait(ctx, port.RunHandle) (port.RunResult, error)`
  - `Stop(ctx, port.RunHandle) error`
  - `SupportedCapabilities() model.CapabilitySet`
  - ⚠️ DTOs : `RunSpec`/`RunHandle`/`RunResult` dans le package **`port`** ; `CapabilitySpec`/`CapabilitySet`/`ProvisionResult` dans **`model`** (divergence avec le plan directeur §1.2 qui écrit `model.RunSpec` — **suivre le code**). `RunSpec` ne porte aujourd'hui **ni réseau, ni sidecar, ni healthcheck, ni limites** (incomplet → extension en P3c).
- **`port.ContainerManager`** (+ `port.SidecarManager`) = ports **réellement utilisés** par le flow live (`agent_run.go`), Docker-shaped. Le live n'utilise pas `AgentRuntime`.

## Intégration microsandbox (recherche — à re-vérifier sur le source avant P3b)

- microsandbox = runtime microVM **libkrun**, Apache-2.0, **BETA v0.5.8** (éditeur Super Rad Company). **KVM requis** sur Linux (HVF macOS Apple Silicon, WSL2 Windows).
- **SDK Go officiel** : `github.com/superradcompany/microsandbox/sdk/go`. Protocole : JSON-RPC 2.0 sur HTTP (`/api/v1/rpc`), auth JWT, serveur `msb server` ; exec forwardé au `microsandbox-portal` interne à chaque VM.
- Modèle prod aligné hexagonal : `msb server start` (JWT `msb server keygen`) par nœud d'exécution ; l'adapter Go = client (`server_url` + `api_key`).
- Surface SDK Go rapportée (à confirmer sur source) : `CreateSandbox(WithImage/CPUs/Memory/Env/Network/Secrets/...)`, `Exec/Shell/ShellStream` → `Stdout/Stderr/ExitCode` (exit≠0 **n'est pas** une erreur Go), `Stop/Kill/RemoveSandbox`, `ListSandboxes`, erreurs typées.
- **Clone CoW golden images = point fort confirmé** : rootfs 2 block-devices (lower EROFS read-only content-addressed par diff-ID partagé entre sandboxes + upper ext4 overlayfs par sandbox) → clone instantané. On bake les stacks du catalogue P2a en images OCI → fit direct.
- **agent-runtime (claude/opencode CLI) reste substrat-agnostique → aucun changement requis** : l'adapter lance le même binaire dans la microVM.

**Inconnues à lever sur le source AVANT P3b** : présence réelle de `server_url`/`api_key` côté Go ; port serveur (5555 docs vs 6765 quick-start) ; modèle embedded « no daemon » (README Go) vs serveur HTTP ; **topologie réseau multi-sandbox** (sidecars communicants par nom) — non documentée, modèle par-sandbox ≠ « bridge multi-VM » Docker ; testcontainers/DinD natifs dans la microVM (plausible, non confirmé) ; commit/snapshot live VM→image (non documenté — golden = bake OCI en amont, pas snapshot).

## Découpage

### P3a — Squelette + sélection par config (MERGÉ, sans KVM, build/test-only, 0 régression Docker)
- `backend/internal/adapter/microsandbox/runtime.go` : `Runtime` + `var _ port.AgentRuntime = (*Runtime)(nil)`. `SupportedCapabilities()` (données) + `Provision()` (tri agnostique→supporté, **warn+skip**) + helper pur `ResolveImage` (StackRef sinon image). `Launch/Wait/Stop` = stubs `ErrNotImplemented` (réel = P3b).
- Config : `SubstrateConfig{Kind}` (défaut **`docker`**) + override env `SUBSTRATE` + validation (`docker`|`microsandbox`).
- `main.go` : fabrique `selectSubstrate` qui **logue** le choix ; `docker` → comportement inchangé (nil) ; `microsandbox` → adapter inerte + Warn « scaffold-only (P3a): live execution still uses Docker ». **Ne touche pas** ContainerManager/agent_run/sidecars.
- `go.mod`/`go.sum` inchangés (aucune dépendance externe — le vrai SDK vient en P3b).

### P3b — Appels microVM réels (besoin host KVM, NON validable en CI/devcontainer)
- `Launch/Wait/Stop` via le SDK Go microsandbox (mode `msb server` + client) : `CreateSandbox(WithImage(stack), WithCPUs/Memory, WithNetwork, WithSecrets)`, `Exec/ShellStream` pour lancer agent-runtime, streaming logs, récolte exit code, teardown (`Stop`/`RemoveSandbox` + GC via `ListSandboxes`).
- Clone CoW des golden images (stacks catalogue bakées OCI).
- **Réseau isolé par run + topologie sidecars** : non documentée → trancher (1 microVM multi-process vs N sandboxes + ports/policies) pour conserver la parité avec le réseau isolé + DNS du ContainerManager Docker.
- Pin de version stricte du SDK (beta). Validation par vrai run (+ testcontainers/DinD natifs).

### P3c — (Optionnel) Bascule du flow live vers `AgentRuntime`
- Étendre `port.AgentRuntime`/`RunSpec` (réseau isolé, sidecars, healthcheck, limites CPU/mem — aujourd'hui absents, présents dans `model.ContainerOpts`).
- Faire implémenter `AgentRuntime` aussi par l'adapter Docker, puis migrer `agent_run.go` + OrphanCleaner/TimeoutEnforcer/SidecarGC de `ContainerManager`/`SidecarManager` vers `AgentRuntime` sans toucher au domaine.

## Décisions ouvertes (utilisateur)

- **Cible de déploiement KVM** : OpenShift avec `/dev/kvm` exposé (KubeVirt) ? bare-metal ? cloud nested-virt (AWS C8i/M8i/R8i) ? Détermine la faisabilité de P3b et l'invariant Pod-exprimable.
- **Version microsandbox** à pinner (beta v0.5.8, breaking changes annoncés).
- **Mode d'intégration** : serveur (`msb server` + client JSON-RPC/SDK Go, JWT — plus probable, aligné hexagonal) vs embedded « no daemon ». À trancher sur le source.
- **Fallback si KVM absent** alors que `SUBSTRATE=microsandbox` : (a) warn + stub not-implemented (comme `containerMgr==nil`) ; (b) fallback auto vers Docker ; (c) fail-fast au boot. Recommandé (a) en P3a, décider (b/c) pour P3b.
- **Topologie sidecars sous microVM** (non documentée) : 1 microVM multi-process vs N sandboxes + ports.
- **Golden images** : bake les stacks du catalogue P2a en OCI spécifiques microsandbox (clone CoW) maintenant, ou réutiliser les mêmes digests que Docker ?

## Risques

- microsandbox **BETA** : breaking changes, features manquantes. Mitigation : isolation stricte derrière le port + pin de version.
- Exactitude du SDK Go **non vérifiée sur source** (README lu via fetcher). Avant P3b : lire le vrai source `sdk/go` + `references/api`.
- **Topologie réseau multi-sandbox** non documentée → divergence potentielle vs le modèle Docker (réseau isolé par run + DNS). Risque sur l'invariant sidecars-in-Pod.
- testcontainers/DinD natifs microVM **non confirmés** officiellement — pourtant argument clé de la décision #14.
- Contrainte d'infra forte : **KVM/nested-virt requis**. Sans cible KVM concrète, P3b reste bloqué et P3a n'est jamais exercé en conditions réelles.
- Absence de champ réseau/healthcheck/limites dans `RunSpec` et de Volumes/mounts dans le modèle → extension nécessaire en P3b/P3c pour le clone CoW + les golden images.
