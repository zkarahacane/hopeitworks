# Board & Planning Connectors — Functional Review

> Lens: product & user value, not engineering. The engineering is assumed sound. This challenges *what we build first and why*, *whether the flows match the job-to-be-done*, and *where the plan optimizes for adapter coverage over adoption*.

---

## 1. Verdict (TL;DR)

- **The architecture is right; the priority order is backwards for this product.** The plan leads with GitHub ("highest demand") and treats the BMAD/markdown rich path as a P0 *parser fix* rather than the headline flow. But the product's own dogfooding story, its differentiation, and its lowest-friction "plan anywhere → it just runs" promise all live in the markdown/BMAD path. **That should be the first complete, demoable connector — not a cleanup task buried inside P0.**

- **The plan is an *import* plan, not a *connection* plan.** It defines exquisite field-mapping tables and an idempotent upsert, but the **first-run experience of connecting a source is undefined**: there is no auth/onboarding flow, no "pick what to import" scoping, no preview-before-commit, no empty-state. A consultant's first 5 minutes with a connector is the entire adoption moment, and it is absent.

- **The enrichment + HITL gate is a strategic risk to the core promise.** "Plan anywhere → execute here" sells *frictionless*. Inserting an LLM enrichment round-trip plus a human review gate between import and run — for *every thin issue* — directly contradicts that promise. Enrichment is real value, but framed as a *mandatory wall* it becomes the thing that makes the product feel like work. It must be opt-in / async / overridable, not a gate every story must pass.

- **The rich-path positioning is correct but under-sold.** "Rich spec → launch-ready, no scoping, no HITL" is genuinely the killer flow and the plan says so (§5.3). But it's positioned as the *exception* (the happy accident when a spec happens to be rich) rather than as **the recommended way to use the product**. The product should actively *pull* users toward writing richer specs, because that's where success rate is highest.

- **Critical functional scenarios are unspecified:** multi-source in one project, what happens when an external issue changes mid-run, cross-source dedup, who is allowed to connect a source, and the demo/sales narrative. Several of these aren't "later" — they're load-bearing for whether a consultant trusts the tool with a real client backlog.

- **Net:** strong skeleton, correct one-way-first instinct, but it reads as a backend roadmap that will produce *traceable imports* before it produces an *adopted product*. Re-sequence around the user's first win, make enrichment optional, and lead with the path the team already lives in.

---

## 2. What the plan gets right (functionally)

- **One canonical internal story, N sources.** Decoupling planning from execution and normalizing everything to one `model.Story` is exactly the agnostic-execution-layer thesis. This is the correct product spine.
- **One-way-first.** Refusing to build write-back, conflict resolution, and webhook infra in v1 is the right product call: 90% of the value (visibility + agent-ready stories) at a fraction of the risk. Good discipline.
- **Provenance as a first-class concern.** Recognizing that a story must carry `source` + `external_id` + `source_url` — and that the current "Planned in" badge is a lie derived from `git_provider` — is a real user-trust fix. A consultant managing 4-5 clients *needs* to know "this card came from client X's Jira, here's the link." Source badge + deep-link is genuine value.
- **The rich-path insight itself.** Naming that a BMAD `ready-for-dev` story is "the plan, not a pointer to a plan" and therefore skips scoping is the single best product idea in the document. The instinct is right even if the positioning isn't.
- **Idempotent re-sync.** "Re-sync is always safe" is a quietly important UX property — it means the user can mash the Import button without fear, which is exactly the confidence a busy consultant needs.
- **Not mirroring an external board.** The decision that the in-app kanban is *generated*, not a GitHub Projects mirror, is correct — it keeps the live execution tree (the actual differentiator) as the centerpiece instead of reducing the product to a second-rate Jira view.

---

## 3. Functional gaps & risks

### 3.1 The connection flow doesn't exist (biggest gap)
The plan jumps from "adapter exists" to "stories appear." Missing the entire middle:
- **Auth onboarding.** How does the user authorize GitHub App / GitLab token / Jira OAuth? Where does that live — project settings? A connector wizard? This is multi-step OAuth dancing that the plan waves at ("Auth: GitHub App") without any UX. For a consultant, *connecting client Jira* is the scariest, highest-friction moment (permissions, security, "what will this tool touch?").
- **Scoping the import.** A real Jira project has 800 issues. The plan implies "Fetch → upsert all." Nobody wants 800 cards. **There is no "import which board / which epic / which label / which sprint" selector.** This is not polish; without it the first import is unusable.
- **Preview before commit.** No "here's what we'll create (12 epics, 47 stories) — confirm?" step. First import should never be a silent bulk-write into the user's board.
- **Empty state / first-run.** What does a brand-new project with no source connected show? What's the call-to-action? Undefined.

### 3.2 Enrichment-as-a-gate fights the promise
- The flow `import → enrich (LLM) → needs_review HITL → run` means a thin GitHub issue **cannot just run**. For the "I planned in GitHub, now execute" user, this is friction exactly where the product promised magic.
- `ambiguity_score > 0.4 → needs_review` is an **arbitrary, invisible threshold** making an adoption-critical decision. Who tuned 0.4? A user whose stories keep landing in "needs review" will conclude the tool doesn't trust them and churn.
- **Missing:** the ability to *skip* enrichment ("run it raw, I'll deal with it"), to *bulk-approve*, to *auto-approve below a confidence bar*, or to enrich *on launch* rather than *on import* (so import is always instant and the gate only appears at the moment of running). The plan hard-wires enrichment into ingestion; it belongs at execution time, opt-in.

### 3.3 The rich path is positioned as exception, not default
- §5.3 treats `Enriched==true` as a lucky fast-path. Product-wise it should be the **promoted, documented, first-class way to get the best results** — with the product nudging users up the richness ladder (thin issue → "want better results? add acceptance criteria" → rich spec → launch-ready).
- No notion of a **richness/readiness score on the card** that tells the user *why* a story will or won't run well. That feedback loop is what would actually move users toward writing better specs — the highest-leverage thing for success rate.

### 3.4 Multi-source & dedup undefined
- Can one project pull from *both* a GitHub repo and a Jira project? A consultant might. What happens to `key` collisions, ordering, epic merging? Unspecified.
- **Cross-source dedup:** the same work item mirrored in Jira *and* GitHub (common in enterprises) imports as two cards. No dedup story.

### 3.5 Mid-run external change
- Story imported → enriched → agent running → someone edits the Jira ticket. Re-sync upsert would clobber the in-flight story's fields (the `COALESCE(enriched_spec)` guard protects the *enriched* blob but not title/AC/objective). **No "locked while running" concept.** This is a correctness-of-experience issue, not just data.

### 3.6 Bidirectional expectation gap
- Users connecting Jira/GitHub will *expect* their ticket to move when the agent finishes — that's the whole point of a tracker. Write-back is P2/v3. That's defensible for build order, but the **product must set the expectation explicitly** ("import-only for now; your Jira won't update") or users feel the integration is broken. This is a messaging gap, not just a feature gap.

### 3.7 Permissions / who-can-connect
- Connecting a source means handing the platform credentials to a client's tracker. **No model for who in a project can connect, what scopes are requested, or how a consultant explains this to a client's security team.** For the target user (multi-client consultant), this is a *sales blocker*, not a detail.

### 3.8 No demo/sales narrative
- What's the 3-minute demo? The plan has no "golden path" story. The most compelling one — *write a markdown spec, drop it in, watch agents build it live* — is exactly the rich path the roadmap de-prioritizes.

---

## 4. Value-based re-prioritization (vs the plan's P0/P1/P2)

The plan ranks by **foundation-then-demand** (traceability → GitHub → others). I re-rank by **time-to-first-win and differentiation**. Reasoning per move below.

### NEW P0 — "Plan in markdown, watch it build" (the dogfood + demo flow, end-to-end)
Make the **markdown/BMAD path a complete, demoable product**, not a parser fix.
- Source traceability fields (`source`/`external_id`/`source_url`) + idempotent upsert — *kept from plan P0, it's the right foundation.*
- **Fix + promote the markdown adapter** to a real connector: stop discarding `epic`/`objective`/`target_files`, set `Enriched=true` for `ready-for-dev`, **and give it a real entry UX** (drop a folder / paste markdown / point at a repo path → preview → confirm).
- Thicken `buildClaudeMD` so even a minimal prompt is grounded — *kept from plan P0, it's a cheap win that lifts every story's success rate.*
- Provenance badge + deep-link on the card — *pull from plan P1; it's cheap and it's what makes the board trustworthy.*

**Why first:** This is the path the team *already lives in*, it needs **no OAuth, no enrichment, no HITL**, it is the lowest-friction expression of "plan anywhere → it just runs," and it is the best demo. It proves the whole thesis with the least surface area. Shipping this is the product's first real "wow."

### NEW P1 — First external source, *with a real connection UX*, enrichment optional
- **GitHub adapter** — *kept as first external source; demand is real.* But the deliverable is the **connection flow**, not just the adapter: auth onboarding, **import scoping** (which repo/project/label), **preview-before-commit**.
- Connector port + import service — *kept from plan P1.*
- **Enrichment as an opt-in, execution-time step — NOT an ingestion gate.** Reframed from plan P1: import is always instant; enrichment runs *when the user launches* a thin story, or as a one-click "enrich this card." Auto-approve below a confidence bar; bulk-approve; "run raw" escape hatch. The HITL gate becomes a *choice*, not a wall.
- **Richness/readiness indicator** on every card (why this will/won't run well) — *new; the feedback loop that moves users up the spec-quality ladder.*

**Why second:** External demand justifies GitHub, but the *value* is in the connection experience and in **not** breaking the frictionless promise. The adapter is the easy 30%; the onboarding + optional-enrichment framing is the 70% that decides adoption.

### NEW P2 — Breadth, real-time, and closing the loop
- **GitLab + Jira adapters** — *kept from plan P2.* Breadth matters for the multi-client consultant but only after the GitHub flow proves the connection UX generalizes.
- **Write-back** — *kept at P2, but elevate its messaging earlier:* until it ships, the UI must say "import-only." When it lands it's a real differentiator ("your tracker updates itself").
- **Webhooks / auto-sync** — *kept from plan P2.* Nice-to-have until a user has lived with manual re-sync and asked for it.
- **Multi-source in one project + cross-source dedup** — *new explicit item;* defer the build but **decide the product stance now** so the data model doesn't paint us into a corner.
- Board polish (depends_on chips, labels/priority filters) — *kept from plan P2.*

### What moved and why (summary)
| Item | Plan | Here | Reason |
|---|---|---|---|
| Markdown/BMAD **as a product flow** | P0 (parser fix) | **P0 (headline)** | Dogfood + best demo + zero-friction; it *is* the thesis |
| Provenance badge on card | P1 | **P0** | Cheap; makes the board trustworthy on day one |
| GitHub adapter | P1 | **P1** | Demand real, but it's the connection *UX* that matters |
| **Connection/onboarding flow** | (absent) | **P1 (explicit deliverable)** | The adoption moment; currently undefined |
| Enrichment + HITL | P1 (ingestion gate) | **P1 (opt-in, execution-time)** | A mandatory gate breaks "it just runs" |
| Readiness/richness score | (absent) | **P1 (new)** | Feedback loop that lifts success rate |
| GitLab/Jira | P2 | P2 | Correct; breadth after the pattern proves out |
| Multi-source / dedup stance | (absent) | **P2 decision now** | Data-model risk if ignored |

---

## 5. Top product questions the team must answer

1. **Is the markdown/BMAD path the flagship or a feature?** If the team plans in BMAD and that's also the best demo and the lowest-friction flow — commit to it as *the* recommended way to use the product, and build the roadmap around it. (Recommendation: yes, make it the flagship.)

2. **Is enrichment a gate or a choice?** Decide now whether a thin issue can run *raw, immediately* with enrichment as an optional/auto/execution-time step — or whether every thin story must pass an LLM + human gate. This single decision determines whether the product feels like "plan anywhere → it just runs" or "plan anywhere → then do homework." (Recommendation: choice, defaulted to instant.)

3. **What is the connection flow, concretely?** Define the first-run experience for connecting a source: auth, **import scoping** (which board/epic/label — not "everything"), preview-before-commit, empty state. This is the adoption moment and it currently doesn't exist on paper.

4. **What do we promise about write-back, and when do we say it?** Users will expect their Jira to move. Until v3, the product must explicitly message "import-only." Decide the messaging now, not when users file "the integration is broken" tickets.

5. **What's the security/permissions story for connecting a *client's* tracker?** For a multi-client consultant, "what scopes does this request and how do I justify it to the client's security team" is a sales gate. Who can connect, what's requested, and how it's explained — needed before Jira/GitLab land in a real client engagement.
