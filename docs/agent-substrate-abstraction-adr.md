# ADR — Agent substrate abstraction: `port.AgentRuntime` as the one execution path, Docker as an adapter

- **Status:** Proposed
- **Date:** 2026-06-23
- **Deciders:** runtime/platform
- **Supersedes for the runtime layer:** the incremental "branch-beside-Docker" approach prototyped in the `p3c` worktree (`.claude/worktrees/p3c/backend/internal/adapter/action/agent_run_substrate.go`).
- **Builds on:** `docs/agent-runtime-capabilities-plan.md` (port generalisation §1.2, substrate-per-context §12–14, image-substring kill §1.3).

> Convention note: `docs/` has no formal ADR directory or numbering scheme (flat `*.md`, plan/research files). This ADR follows that flat convention rather than inventing an `adr/NNNN-` tree.

---

## 1. Context

### The realisation

An agent is fundamentally **"exec a harness"**: clone the repo, run `agent-runtime`, which drives `claude -p` / `opencode` (provider chosen by `PROVIDER` env). The agent reports its outcome back over plain HTTP. *Where* that exec happens — a Docker container, a bare `exec` process, a microVM — is a **separate, pluggable concern**. Docker is **incidental, not intrinsic** to running an agent.

The evidence is already in the tree. The agent→API callback client is a stdlib `*http.Client` POST/GET to a base URL with zero substrate awareness (`agent-runtime/internal/callback/client.go`). Any substrate that can (a) run the `agent-runtime` binary and (b) give it outbound HTTP to the API satisfies the entire contract. The substrate is invisible to the outcome.

### Current state (develop, b964795)

- **`agent_run` is Docker-coupled.** `AgentRunAction` holds `containerMgr port.ContainerManager` (`backend/internal/adapter/action/agent_run.go:36`) and drives it directly: `createContainer` → `containerMgr.Create` (`agent_run.go:442`), `containerMgr.Start` (`agent_run.go:224`), `cleanupContainer` Stop/Remove (`agent_run.go:531-546`). The Docker coupling is *concentrated*, not pervasive: `ContainerManager` is touched in exactly five logical sites (`Create` `:442`, `Start` `:224`, cleanup `:535/:540`, `LogStreamer` in `streamAndWait` `:454`, and the ephemeral env-commands in `agent_run_commands.go`). Everything else — prompt, env, CLAUDE.md, conn-strings, token, cost-record, outcome→error — is already substrate-agnostic logic that merely lives inside Docker-named methods.
- **Two divergent outcome paths.** `Execute` branches on `isCallbackMode` (`agent_run.go:233-237`):
  - **Callback mode** (`waitForCallback`, `agent_run.go:639-665`): outcome comes *entirely* from the HTTP callback channel — `statusStore.WaitForStatus(ctx, stepID, 2h)` (`:642`), filled out-of-band by `backend/internal/api/handler/agent_callback_handler.go:163` (`SetStatus`). The container's own exit code is never consulted. **Already substrate-agnostic in spirit.**
  - **Legacy mode** (`streamAndWait`, `agent_run.go:446-527`): outcome = container exit code read off the Docker log streamer (`exitCode := <-doneCh`, `:500`), with `ctx.Err()` disambiguation (`:501-505`) and cost scraped from log lines (`logEvent.Type == "cost"`, `:482-489`). **Docker-specific.**
- **The target port exists but is inert.** `port.AgentRuntime` (`backend/internal/domain/port/agent_runtime.go:44-62`) is the intended seam (`Launch`/`Wait`/`Stop`/`Provision`/`SupportedCapabilities`). The port that supersedes `ContainerManager` is `backend/internal/domain/port/agent_runtime.go:9-15`; `docs/agent-runtime-capabilities-plan.md:71` documents that "ContainerManager reste une dépendance interne de l'adapter substrat" once the migration completes.
- **P3a/P3b microsandbox adapter, merged, behind a build tag.** The only real implementation of `AgentRuntime` is microsandbox: pure logic untagged (`microsandbox/runtime.go` — `SupportedCapabilities`, `Provision`, `ResolveImage`), the real `Launch`/`Wait`/`Stop` behind `//go:build microsandbox` (`runtime_microsandbox.go:119/166/190`), and no-op stubs for the default build (`runtime_stub.go:27/32/37`). It is constructed by `selectSubstrate` (`cmd/api/main.go:589`) but **NOT wired into the live path** — `main.go:594` warns it "constructed but not wired into live agent_run; live execution still uses Docker."
- **P3c "branch-beside-Docker" attempt, parked (uncommitted).** In the `p3c` worktree, `WithAgentRuntime` adds an optional `runtime` field and `Execute` forks: `if a.runtime != nil { return a.executeOnRuntime(...) }` else the Docker path. This is the incremental anti-pattern (see §6) — treated here as a *reference*, not the target.

### The product need: substrate-per-context

The capabilities plan (`docs/agent-runtime-capabilities-plan.md` §12–14, lines ~36–40, 248–256) decided three substrates: **microsandbox** (libkrun microVM, the documented default once KVM is available), **K8s + gVisor** (OpenShift without KVM), and **CMA-cloud** (remote managed). It committed to microsandbox-first and generalising `ContainerManager` into `AgentRuntime` (§1.2), with `ContainerManager` kept as "une dépendance interne de l'adapter substrat, pas un port de domaine".

**This ADR adds two substrates not in the capabilities plan** — made possible by the abstraction itself. The point is **N adapters behind one port**, all equal; *which* one runs is a **deployment policy** chosen by config (`SUBSTRATE`), not a hard-coded special case. Per **Decision #3 (§7)**, the **current prod default/requirement is microsandbox** (untrusted agent code needs kernel-class isolation); `docker`/`exec` are dev/CI fallbacks (no KVM); and **a K8s/OpenShift adapter (RuntimeClass gVisor/Kata) is a future extension (P4)** this abstraction makes addable without touching the domain:

| Substrate | Role | Selected when |
|---|---|---|
| **microsandbox** (libkrun microVM) | **Current prod default/requirement** (Decision #3, policy) — true kernel isolation for untrusted agent code; capabilities-plan §14, decided 2026-06-22; testcontainers/DinD inside | `SUBSTRATE=microsandbox`, KVM host |
| **Docker** | Dev/CI fallback (this ADR) — works wherever a daemon exists; the back-compat adapter that keeps the refactor safe; not for prod untrusted code | `SUBSTRATE=docker`, no KVM |
| **exec** (no container) | Local inner-loop fallback (this ADR) — zero Docker dependency, fastest loop, in-process debuggable; not for prod | `SUBSTRATE=exec`, local dev |
| **K8s/OpenShift** (RuntimeClass **gVisor**/**Kata**) | **Future extension (P4)** — addable behind the same port when an OpenShift client arrives, **no domain refactor**; invariant: stay **Pod-expressible** (capabilities-plan §14) | future, `SUBSTRATE=k8s` |

**This ADR is the concrete architecture that delivers the port generalisation: any number of substrates behind one port, the live one chosen by policy. microsandbox is today's prod default — not a special case — and the K8s/OpenShift door stays open precisely because the abstraction keeps every adapter equal.**

---

## 2. Decision

### (a) `port.AgentRuntime` is THE way agents run

There is **one** execution path. Docker stops being special — it becomes a `DockerRuntime` adapter implementing `port.AgentRuntime`, a sibling of `exec` and `microsandbox`. The `if a.runtime != nil` fork from P3c is explicitly rejected (§6): the Docker path must *go through* the port, not sit beside it. `ContainerManager` is demoted to an internal dependency of `DockerRuntime` (the port that supersedes it is defined at `backend/internal/domain/port/agent_runtime.go:9-15`; `docs/agent-runtime-capabilities-plan.md:71` states "ContainerManager reste une dépendance interne").

### (b) `agent_run` (Action) becomes substrate-agnostic

The Action builds an agnostic `RunSpec`, dispatches via `AgentRuntime.Launch`, and **owns the callback-wait + token-mint/revoke once for all substrates**. Today these are stranded in the Docker-only `waitForCallback` (`agent_run.go:639-665`) and the Docker-only `createContainer` token mint (`agent_run.go:383-393`). They move up.

**Token detail:** the token is minted per agent/role — `tokenStore.Create(ctx, runID, stepID, agentID, role, 2h)` at `agent_run.go:388`; the Action mints it from `runCtx` and then **fills** `CallbackSpec.AuthToken` with the result. `CallbackSpec` is not the input to the mint — it is the downstream carrier. Note also that token revoke is currently **dead**: `findContainerToken` (`agent_run.go:669-673`) always returns `("", false)`, so the revoke at `:652` never fires and the token expiry relies entirely on the 2h TTL. Stage 3 fixes this by threading the minted token through so revoke actually fires — a net security gain.

Agnostic logic that already exists and stays in the Action: fetch story/project (`agent_run.go:118/124`), render prompt + retry context (`:130-161`), resolve image selection (`:164`), mode detection from `runtime_kind` (`:170`, `isCallbackMode:622-628`), resolve Environment (`agent_run_environment.go:21`), build conn-strings (`agent_run_environment.go:45-130`), build agent env (all groups `agent_run.go:342-418`), `buildClaudeMD`/`priorFailureContext` (`:257/:689`), API-key resolve (`:369-376`), labels (`:426-431`), exit→error mapping (`:242-247`).

### (c) Docker becomes a `DockerRuntime` adapter (no special path)

`DockerRuntime.Launch(RunSpec)` builds `model.ContainerOpts` from the spec (reusing the `buildAgentEnv`/`buildAgentLabels` extraction prototyped in P3c), `Create`+`Start`s, returns the container id as `RunHandle.ID`. `Stop` wraps cleanup. Critically, it must emit **byte-identical** `ContainerOpts` for the no-Environment case so the existing golden (`agent_run_test.go:1146-1227`) stays the regression oracle (verified green on develop).

### (d) The callback channel is the agnostic source of truth for outcome (and cost)

For callback-mode runtimes (claude_code/opencode/cma — `agent_run.go:626-628`), the outcome already arrives over HTTP (`SendStatus` → `SetStatus` → `WaitForStatus`) independent of any substrate. The Action calls `Wait` only to know the exec *finished* (crash detection); the **authoritative** exit code/error comes from `statusStore.WaitForStatus`. **Outcome reconciliation rule:** if the adapter `Wait` returns non-zero but no callback status arrives (POST exhausted after retries, `agent-runtime/internal/callback/client.go:195-236`), the Action treats this as a crash error — "adapter Wait non-zero without callback status ⇒ crash error". The reverse (callback delivers a status but Wait never returns) is bounded by the 2h timeout (`agent_run.go:642`).

**Cost is substrate-agnostic via the callback handler, and the callback carries the real provider cost.** `HandleCost` (`backend/internal/api/handler/agent_callback_handler.go:110-147`) records cost server-side via `costSvc.RecordStepCost` (`:144`), independent of the Action, for every substrate. Per **Decision #1 (§7)**, the callback becomes **cost-first**: `CostCallbackRequest` (`agent_callback_handler.go:23-27`) and `model.CostEvent` (`cost_record.go:35-39`) gain a `cost_usd` field; the agent-reported provider/CLI cost (`agent-runtime/internal/callback/client.go` `costPayload.CostUSD`) is persisted as-is when present, and **server-side pricing-table re-derivation (`cost_service.go:44`, `cost_record.go:48`) stays as the fallback** for harness/CLI that don't report it. So the authoritative source is the callback (provider-real when available, derived otherwise) — no special path per substrate. The only Docker-specific cost path is legacy log-scraping (`agent_run.go:482-489/:518-524`), which dies with legacy mode. **Live-log SSE does not depend on the Docker log stream either** — SSE is sourced from the Postgres event bus (`sse_handler.go`), fed identically by both `publishLogEvent` (legacy) and `HandleLogs` (callback, `backend/internal/api/handler/agent_callback_handler.go`), and `log_tail` is persisted in both modes. So the callback channel + Postgres event bus already carry outcome, **cost (provider-real)**, live logs and `log_tail` with zero Docker coupling.

### (e) exec + microsandbox are sibling adapters

Once the Action dispatches via the port and the outcome flows from the callback, **exec** runs `agent-runtime` as a child process with the callback env quartet (`agent_run.go:395-400`; no isolation, for local dev) — it needs Stage 4 reaper work before it is production-safe, but is appropriate for local dev and **microsandbox**'s already-merged `Launch`/`Wait`/`Stop` (`runtime_microsandbox.go:119/166/190`) become live without any special-casing in the Action.

### Target port shape

`RunResult` stays minimal because cost+outcome flow over the callback. `RunSpec` grows the fields the live Docker path actually needs but the current spec lacks.

```go
// port/agent_runtime.go  (target)
type RunSpec struct {
    // identity / harness (exist today, agent_runtime.go:19-28)
    RuntimeKind  string
    Model        string
    Provider     string
    Image        string            // stack-key-or-ref; the ADAPTER resolves catalogue digests
    Prompt       string
    Env          []string          // KEY=value; conn-strings live here (pure, agnostic)
    Labels       map[string]string // run_id/step_id/story_key — agnostic bookkeeping
    Capabilities model.CapabilitySpec

    // NEW — resources (today read from AgentConfig at agent_run.go:423-424)
    Memory int64
    CPUs   float64

    // NEW — agnostic connectivity (replaces leaking sidecarCtx.NetworkName at agent_run.go:438-440)
    Network RunNetwork

    // NEW — one-shot overrides (folds the ephemeral env-commands at agent_run_commands.go:177
    //        onto the SAME port so build/migrate/seed run on every substrate)
    Entrypoint []string
    Cmd        []string
    Workdir    string

    // NEW — callback contract as typed fields, not opaque Env (today stuffed at agent_run.go:392-400)
    Callback *CallbackSpec
}

type RunNetwork struct {
    Attachments []ServiceEndpoint  // derived from SidecarContext.ServiceAddrs
    Aliases     map[string]string
}
type ServiceEndpoint struct { Name, Host string; Port int }
type CallbackSpec struct { URL, AuthToken string; RunID, StepID uuid.UUID }

type RunHandle struct { ID string }   // container/microVM/pid

type RunResult struct {
    ExitCode int
    Error    string
    Cost     []model.CostEvent // OPTIONAL; populated only by non-callback (legacy) adapters
}

// AgentRuntime: unchanged shape — Provision / Launch / Wait / Stop / SupportedCapabilities
```

**Do not** put a raw `NetworkName string` on `RunSpec` — that re-leaks the Docker abstraction P3c warned about. Docker maps `RunNetwork` to `ExtraNetworks`+aliases; microVM maps it to host routing or degrades to conn-strings-in-`Env` (which already work, `agent_run_environment.go:45-130`).

### What stays in the Action vs the adapter (the seam)

| Concern | Citation | Owner in target |
|---|---|---|
| Prompt / env / CLAUDE.md / conn-strings / labels | `agent_run.go:130-161,342-431`, `agent_run_environment.go:45-130` | **Action** → `RunSpec` |
| Mode detection (`runtime_kind`) | `agent_run.go:170,622-628` | **Action** (harness property) |
| Token **mint** + **revoke** | `agent_run.go:383-393`, `:648-657` | **Action**, once, bracketing dispatch, all substrates |
| Callback-wait (outcome) | `agent_run.go:639-665` | **Action**, once, all substrates |
| Environment **orchestration** (build→migrate→seed→test order, fail-fast, readiness-before-agent) | `agent_run_commands.go:82-125`, `agent_run.go:212` | **Action** (run-lifecycle policy, substrate-neutral) |
| Environment command **exec** | `agent_run_commands.go:149-213` | **Adapter** (one-shot `RunSpec` with `Entrypoint`/`Cmd`) |
| Sidecar **launch/teardown** | `agent_run.go:186,194-199` | `SidecarManager` stays **action-level**; substrate owns only *attachment* via `RunNetwork` |
| Create / Start / Stop / Wait | `agent_run.go:442,224,531-546` | **Adapter** `Launch`/`Stop`/`Wait` |
| Legacy log-stream + exit + log-cost | `agent_run.go:454,500,482` | **Adapter** internal (DockerRuntime only), or retired with legacy mode |

**On sidecars/Environment specifically:** keep *orchestration* in the Action and let the substrate own only *attachment/connectivity*. Sidecars are already action-orchestrated through a separate `SidecarManager` port (`docker/sidecar_manager.go` is built *on top of* `ContainerManager`, not vice versa). The Action — not Docker — decides ordering, readiness, conn-string injection and teardown. Pushing that into each substrate would force every adapter to re-implement it, and the microsandbox limitation (no shared-network sidecars yet, `runtime.go` doc + P3c scope note) *reinforces* this split: microVM only needs to answer "how do I attach to these endpoints?", degrading to conn-strings-only when it can't.

---

## 3. Consequences

### What gets simpler

- **One outcome path, not two.** The callback channel becomes the single source of truth for outcome+cost across docker/exec/microsandbox, with **provider-real `cost_usd` carried over the callback and the pricing table as fallback** (Decision #1). The `ctx.Err()` disambiguation hack (`agent_run.go:501-505`), the log-cost scrape (`:482-489`), and the per-mode cost asymmetry all disappear when the Docker log-stream mechanism retires.
- **Logs + prompt stay auditable, substrate-free.** Per Decision #2, retiring the Docker log-stream removes only the *mechanism*; logs and the agent prompt persist through the callback path (`HandleLogs` → Postgres bus → `log_tail`/SSE). Audit survives the substrate change.
- **Token lifecycle lives once.** Mint (`agent_run.go:383-393`) + revoke (`:648-657`) move into the Action, bracketing dispatch, identical for every substrate — instead of being split across the Docker container-builder and the Docker callback-wait.
- **The Docker special-case in `main.go` shrinks.** `agent_run` no longer needs `if containerMgr != nil` to *exist* — it needs *a substrate*. The callback stores and `InternalAuth` middleware are already wired unconditionally (`main.go:487`), confirming the agnostic infra is in place.

### What the exec substrate unlocks (local dev)

A bare `exec` adapter — run `agent-runtime` as a child process with the callback env quartet (`CALLBACK_URL`/`AUTH_TOKEN`/`RUN_ID`/`STEP_ID`, constructed at `agent_run.go:395-400`) — gives a **Docker-free local dev loop**: fastest iteration, in-process debuggability, no daemon. This is impossible today because `agent_run` *is* Docker. It becomes a focused adapter once the Action dispatches via the port.

**exec is not a clean freebie though.** It inherits the same reaper blindness as microsandbox: if the child process crashes, `OrphanCleaner` (`service/orphan_cleaner.go:42`) and `TimeoutEnforcer` (`service/timeout_enforcer.go:78`, Stop at `:130`) are blind to it (they list Docker containers), and `CancelRun` (`run_service.go:663`) cannot kill it. The same Stage 4 work that makes reapers substrate-aware applies to exec. This is acceptable for a local-dev-only substrate but must not be forgotten.

### How it fixes the P3c callback gap

P3c's `executeOnRuntime` reads `result.ExitCode` from the adapter's `Wait` (`p3c/agent_run_substrate.go:88`) — re-introducing **per-substrate** outcome and skipping callback-wait + token-revoke for substrates. The target inverts this: outcome flows from the callback channel for **all** substrates, so P3c's gap (substrate runs never wait on the callback nor revoke the token) cannot occur by construction.

### What becomes possible

- **microsandbox as the prod default without special-casing.** Its merged `Launch`/`Wait`/`Stop` (`runtime_microsandbox.go:119/166/190`) become live the moment the Action dispatches via the port — and per Decision #3 this is the **production default**, not an optional add-on: hardened isolation for untrusted agent code with no Action change.
- **Substrate-per-role** per Decision #3 — all equal behind the unchanged port, selected by policy (`SUBSTRATE`): **microsandbox = prod default**, docker = no-KVM CI fallback, exec = local inner-loop, and **K8s/OpenShift (RuntimeClass gVisor/Kata) addable later (P4) with no domain refactor** (Pod-expressible invariant).

### What gets harder / needs care (the hidden 60%)

The out-of-band reapers read the Docker shape directly, NOT via the port: **OrphanCleaner** (`service/orphan_cleaner.go:42` lists `ListContainers(managed_by=hopeitworks)`, reads `run_id`), **TimeoutEnforcer** (`service/timeout_enforcer.go:78` same + `Stop`), **CancelRun** (`run_service.go:663-664` `containerMgr.Stop(*step.ContainerID)`). For Docker-only deployment these keep working unchanged **iff** `DockerRuntime.Launch` preserves the `managed_by`/`run_id`/`step_id` labels and persists a real container id. For non-Docker substrates they are blind (a leaked microVM is invisible to OrphanCleaner). This is real work — and per **Decision #3** (microsandbox is the prod-required substrate) it is **on the critical path to production, not indefinitely deferred**: Stage 4 must land before microsandbox becomes the prod default (see §4 Stage 4). It stays deferrable only for the Docker/exec dev/CI fallbacks.

---

## 4. Staged migration

Principle: each stage is independently shippable and keeps the golden (`agent_run_test.go:1146-1227`, verified green on develop) the regression oracle. Stages 0–2 are additive refactors that keep `ContainerOpts` byte-identical, so the existing Docker-shaped golden guards the whole way; the genuinely risky surface (reapers, legacy retirement) is isolated to Stages 4–5.

**Target after Stages 0–3 (per Decision #3):** the agnostic refactor makes Docker-as-adapter safe, but the destination is **microsandbox as the default substrate for real runs** — Docker/exec remain dev/CI fallbacks. So once the Action dispatches via the port and the callback owns the outcome, **flip the default for production runs to microsandbox** (its `Launch`/`Wait`/`Stop` are already merged, `runtime_microsandbox.go:119/166/190`), keeping Docker as the no-KVM CI fallback. Stage 4 (reapers) is therefore on the critical path to prod, not deferred, *for the microsandbox substrate*.

### Stage 0 — Extract the launch contract (pure refactor, no dispatch)
- **Scope:** pull `buildAgentEnv` + `buildAgentLabels` out of `createContainer` (exactly the reusable artifact from P3c). Docker path still calls `createContainer` → `ContainerManager.Create`.
- **Files:** `agent_run.go` (extract from `:342-432`).
- **Guard:** `TestAgentRunAction_NoEnvironment_GoldenBackCompat` — identical `ContainerOpts`, unchanged.
- **Risk:** trivial. Lowest-risk, highest-leverage step; already prototyped in `p3c`.

### Stage 1 — Introduce `DockerRuntime` implementing `port.AgentRuntime`, NOT on the live path
- **Scope:** new `internal/adapter/docker/runtime.go`; `Launch` builds `ContainerOpts` from `RunSpec` (reuse Stage-0 helpers), `Create`+`Start`, returns container id; `Stop` wraps cleanup; `Wait` wraps `ContainerManager.Wait`/legacy stream.
- **Files:** `docker/runtime.go` (new), `docker/runtime_test.go` (new).
- **Guard:** a *new* golden in the docker package asserting `Launch(RunSpec)` yields byte-identical `ContainerOpts`; the action-package golden still green (action unchanged).
- **Risk:** low; new code, not yet on the live path.

### Stage 2 — Dispatch via the port behind a default-Docker injection
- **Scope:** thread the runtime via an **additive variadic Option** (`WithAgentRuntime`, P3c pattern) so `NewAgentRunAction`'s 17-positional signature (`main.go:303-309`, golden fixture `agent_run_test.go:1147`) stays stable. `Execute` calls `Launch`/`Wait`/`Stop` when a runtime is injected. Wire `main.go` to inject `DockerRuntime` by default; nil falls back to the existing direct path.
- **Files:** `agent_run.go`, `cmd/api/main.go`.
- **Guard:** full suite incl. `orphan_cleaner_test.go`, `timeout_enforcer_test.go`, `run_service_test.go` (label + container-id contract). Golden green: legacy path stays default until the flag is flipped in a controlled deploy.
- **Coverage gap (named, shippable):** the existing action-package golden (`agent_run_test.go:1164`, which observes `f.containerMgr.createCalls` on the mock) does **not** cover the `DockerRuntime` code path all the way to Stage 5. Stages 1–4 cover the new `DockerRuntime` path only via the new docker-package golden introduced in Stage 1. The action golden remains the oracle for `ContainerOpts` invariance. This gap is acceptable for Stage 2 but must be closed before Stage 5 retires the legacy path.
- **Risk:** medium. Mitigated by the additive constructor (no fixture churn) and `ContainerOpts` invariance.

### Stage 3 — Move callback-wait + token-mint/revoke ONCE into the agnostic Action
- **Scope:** after `Launch`, the Action runs `waitForCallback` for ALL substrates in callback mode; **fix `findContainerToken`** (`agent_run.go:669-673`, currently always `("", false)` → revoke is a no-op relying on TTL) by threading the minted token through `RunSpec.Callback`/run-step so revoke actually fires. Legacy `streamAndWait` stays *inside* `DockerRuntime.Wait`.
- **Files:** `agent_run.go`, `port/agent_runtime.go` (add `CallbackSpec`, `Network`, `Memory`/`CPUs`, `Entrypoint`/`Cmd`/`Workdir`, optional `RunResult.Cost`).
- **Guard:** callback-mode tests; golden unchanged (legacy path untouched).
- **Risk:** medium — this is where "callback channel = source of truth" lands. Revoke fix is net-new safety, not a regression.

### Stage 4 — Make the out-of-band reapers substrate-aware
- **Scope:** decide the contract for OrphanCleaner / TimeoutEnforcer / CancelRun. For **Docker-only deployment, zero change** is required as long as Stage 1's `Launch` preserves labels + persists the container id. Non-Docker substrates need a runtime-level reaper/list (or the handle stays a labeled-container id).
- **Files:** `service/orphan_cleaner.go`, `service/timeout_enforcer.go`, `run_service.go` (only if a non-Docker substrate ships).
- **Guard:** `orphan_cleaner_test.go`, `timeout_enforcer_test.go`, `run_service_test.go` green.
- **Risk:** the genuinely risky surface. Per Decision #3, it is **on the critical path before microsandbox becomes the prod default** (a leaked microVM must be reapable); it stays deferrable only for the Docker/exec dev/CI fallbacks. Do not retire the Docker log-stream mechanism (Stage 5) before this is settled, or you lose crash-recovery, user-cancel, timeout-kill.

### Stage 5 — Retire the Docker-specific log-stream mechanism (NOT the logs), optional, last
Per **Decision #2 (§7)**: this stage retires only the **Docker-coupled streaming mechanism**, never the logs. All log retention — and **the prompt, which must stay auditable** — continues through the agnostic callback path (`HandleLogs` → Postgres bus → `log_tail` + SSE).
- **Scope:** drop `streamAndWait` (`agent_run.go:446-527`), the `runtime_kind==""` image-substring fallback (`:630`), the log-cost scrape, and the NDJSON cost parser — i.e. the Docker-stream *mechanism*. **Keep the callback-equivalent log + cost retention intact.** Re-home the golden to assert on `RunSpec` + `DockerRuntime` `ContainerOpts`.
- **Files:** `agent_run.go`, `docker/log_streamer.go`, `docker/ndjson_parser.go`, `agent_run_test.go`.
- **Guard:** re-homed golden in the docker package; **plus an audit assertion that logs + prompt are persisted via the callback path** for a callback-mode run (no Docker stream involved).
- **Risk:** only after telemetry confirms no `runtime_kind==""` runs remain. Audit must not regress: prompts/logs persist via callback before this lands.

### Out of scope (explicitly)

- **Full sidecar-under-microsandbox** (shared run-network inside a microVM). microsandbox supports HTTP MCP / network services but not shared-network sidecars yet; it degrades to conn-strings-only. P3c already scoped this out.
- **Durability polish** for overnight/batch runs beyond what exists (the 2h `WaitForStatus` timeout, `agent_run.go:642`, stays as-is for now).
- **K8s + gVisor adapter** — comes when an OpenShift-without-KVM client arrives (capabilities-plan §14); the port + Pod-expressible invariant make it non-blocking.

---

## 5. Alternatives considered

### A. Branch-beside-Docker (the P3c approach) — rejected as the END state
`WithAgentRuntime` + `if a.runtime != nil { executeOnRuntime } else { dockerPath }` (`p3c/agent_run_substrate.go:34-96`). **Honest upside:** smallest diff, ships a microVM path fast, and its `WithAgentRuntime` additive-Option pattern + `buildAgentEnv`/`buildAgentLabels` extraction are genuinely reusable (we keep them — Stages 0/2). **Why rejected as the end state:** it keeps Docker special and *duplicates the lifecycle* — the Docker branch keeps callback-wait + token-revoke (`agent_run.go:639-665`) while the substrate branch only reads `result.ExitCode` (`p3c/agent_run_substrate.go:88`), so callback-wait and token-revoke **never run for substrates**. Two execution paths, two outcome models, permanent drift. It is the right *increment* (a stepping stone), the wrong *destination*.

### B. Big-bang rewrite — rejected
Rewrite `agent_run` straight onto the port, retire legacy mode, and convert the reapers in one PR. **Upside:** no transitional cruft. **Why rejected:** the golden asserts on `model.ContainerOpts` directly (`agent_run_test.go:1163-1226`); a big bang loses the regression oracle mid-flight, and it couples the safe refactor (Stages 0–3) to the genuinely risky reaper/legacy surface (Stages 4–5) that is gated on real non-Docker need. High blast radius, no incremental shippability.

### C. Keep Docker-only — rejected
Leave `agent_run` Docker-coupled; never add exec/microsandbox to the live path. **Upside:** zero work, zero risk. **Why rejected:** it contradicts the already-made substrate-per-context decision (`capabilities-plan §12-14`), leaves the merged microsandbox adapter permanently inert (`main.go:594`), blocks the Docker-free local-dev loop, and blocks the hardened/multi-tenant story that drives the product. The cost of *not* doing this rises with tenancy urgency.

---

## 6. Risks & mitigations

| Risk | Citation | Mitigation |
|---|---|---|
| **Golden is Docker-shaped** — it asserts on `ContainerOpts`, not behaviour; routing through `RunSpec` changes the asserted type | `agent_run_test.go:1163-1226` | Keep the Action golden asserting `ContainerOpts` until `DockerRuntime` is the thing under test (Stages 0–4); only re-home it in Stage 5. `DockerRuntime.Launch` must emit byte-identical opts. |
| **Overnight / long runs** — 2h callback wait is the only terminal signal in callback mode | `agent_run.go:642` | Unchanged by the migration (same timeout, all substrates). Durability polish explicitly out of scope. |
| **Cost / SSE regression** if legacy `streamAndWait` is dropped before all images report via callback | `agent_run.go:482-524`; `backend/internal/api/handler/agent_callback_handler.go:144` | Cost+SSE+`log_tail` already flow via callback handler + Postgres bus for callback mode. Gate Stage 5 on telemetry confirming no `runtime_kind==""` runs. |
| **Incremental retry quality** depends on `parent.LogTail`/`ErrorMessage` being populated | `incremental_retry.go` reads `LogTail`; populated by `persistLogTail` (legacy) / `AppendStepLogTail` (callback) | Both modes persist `log_tail`; retry is already substrate-agnostic (depends on `AgentRunExecutor` interface, not Docker). Keep `log_tail` populated post-migration. |
| **HITL gate** must keep suspending via the executor re-fetch | `hitl_gate.go` (no Docker); executor suspend-detection | Fully agnostic already; invariant: the Action returns `nil` and leaves step-status mutation to the executor, exactly as today. |
| **Out-of-band reapers blind to non-Docker substrates** | `orphan_cleaner.go:42`, `timeout_enforcer.go:78`+`:130`, `run_service.go:663` | Docker-only path: zero change if labels + container-id preserved (Stage 1). Non-Docker: addressed in Stage 4, gated on a real non-Docker substrate. Guarded by `orphan_cleaner_test.go`/`timeout_enforcer_test.go`/`run_service_test.go`. |
| **Token revoke is currently dead** (`findContainerToken` always false) | `agent_run.go:669-673` | Stage 3 threads the minted token so revoke fires for every substrate — a net safety gain, not a regression. |
| **`RunResult` insufficient for legacy logs/cost** on a non-callback image on a non-Docker substrate | `agent_runtime.go:36-39` | Add optional `RunResult.Cost []model.CostEvent`, populated only by non-callback adapters; Action records it only when non-empty. Callback-mode default stays cost-free in `RunResult`. |

**Guardrail across all stages:** the golden + `WithEnvironment_SidecarWiring` tests (verified green on develop) are the back-compat oracle. No stage merges without both green.

---

## 7. Decisions (resolved)

The four open questions raised during review have been decided. Kept here with short rationale; impact is propagated into §1/§2/§3/§4.

1. **Cost source of truth = the most real cost (provider/CLI `cost_usd`), reported over the callback; server re-derivation is the fallback.** The agent/harness already computes a real `cost_usd` from the provider/CLI (`agent-runtime/internal/callback/client.go` `costPayload.CostUSD`); today `HandleCost` **drops it** and re-derives USD from a pricing table (`cost_service.go:44` / `cost_record.go:48`; `backend/internal/api/handler/agent_callback_handler.go:135-141`). **Decision:** extend `CostCallbackRequest` (`backend/internal/api/handler/agent_callback_handler.go:23-27`) and `model.CostEvent` (`cost_record.go:35-39`) with a `cost_usd` field; persist the reported value as-is when present; **keep server-side pricing-table re-derivation as the fallback** for older harness/CLI that don't report it. This makes the callback the truly agnostic source of truth for outcome **and** cost (provider-real when available, derived otherwise) — no contradiction with §2(d): callback-first, server-derive on absence.
   - *Rationale:* the provider's own number is the most accurate; a pricing table drifts and can't see negotiated/cached pricing. Fallback preserves back-compat with no flag day.
2. **Keep the logs (audit) — retire only the Docker-specific streaming mechanism, never the logs.** Logs, and **in particular the prompt sent to the agent, must be retained for audit.** **Decision:** all log retention flows through the agnostic callback channel (`HandleLogs` → Postgres event bus → `log_tail` + SSE). Stage 5 retires only the Docker-coupled `streamAndWait` log-stream mechanism, keeping the callback-equivalent. **New requirement:** the prompt handed to each agent (`agent_run.go:130-161`) is logged/auditable through the same callback/Postgres path — audit must survive substrate change.
   - *Rationale:* audit (esp. of prompts) is a product requirement; "retire legacy mode" is purely the Docker-stream mechanism, not the data.
3. **Microsandbox is the production default/requirement — as a deployment POLICY, not a reduction of the abstraction.** Strong product decision: **in production the agent runs in a microsandbox (microVM) by default/policy.** This does **not** make microsandbox "the only" substrate or a hard-coded special case — the port stays **fully pluggable** and every adapter is equal; the live substrate is chosen by config (`SUBSTRATE`). `docker`/`exec` remain dev/CI fallbacks (no KVM), not for untrusted prod code. **Crucially, a K8s/OpenShift adapter (RuntimeClass gVisor/Kata) stays addable later (P4, triggered by an OpenShift client) without touching the domain** — the "Pod-expressible" invariant (capabilities-plan §14) is what keeps that door open. **Decision:** after the agnostic refactor (Stages 0–3), **flip the prod default to microsandbox** (policy), keeping Docker/exec as fallbacks and K8s/OpenShift as a future sibling behind the same port.
   - *Rationale:* untrusted agent-generated code needs a real isolation boundary in prod (KVM-class is the floor) — but the architectural win is N adapters behind one port; microsandbox is just the current default, not a special case.
4. **`RunHandle.ID` / `run_steps.container_id` — deferred, with a recommended default (does not block migration).** **Decision (default):** store the substrate handle in `run_steps.container_id` as a **generic opaque id**; CancelRun, OrphanCleaner and TimeoutEnforcer act via that handle plus the `managed_by`/`run_id` labels, resolving `Stop` through the runtime (`run_service.go:663-664`). **Re-attach-by-name after an API restart** (reconnecting to a live substrate handle) is a tracked follow-up, **not** required for the staged migration. The column is not renamed now; if a rename to a generic `runtime_handle` is wanted later it is an additive API/UI change.
   - *Rationale:* the labeled-handle default already satisfies Docker-only reaping (Stage 4) and is substrate-neutral; re-attach is a durability nicety that can land independently.
