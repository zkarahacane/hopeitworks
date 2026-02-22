# Story runtime-1: Agent Docker Image and Entrypoint

**Status:** ready-for-dev
**Branch:** `feat/runtime-1-agent-docker-image`
**Commit scope:** `agent`

---

## Story

As the hopeitworks pipeline executor, I need a Docker image (`hopeitworks/agent:latest`) that can be launched as a container, clone a git repository, check out the correct branch, inject a CLAUDE.md file, execute a Claude Code agent with a rendered prompt, stream structured NDJSON logs back to the platform, and exit with an appropriate code — so that `AgentRunAction` can orchestrate real agent runs end-to-end.

---

## Acceptance Criteria

**AC #1 — Image builds without error**
- Given `agent/Dockerfile` exists in the repository root
- When `docker build -t hopeitworks/agent:latest agent/` is executed
- Then the build succeeds and the image is tagged `hopeitworks/agent:latest`

**AC #2 — Container starts and clones repo**
- Given the image is built and env vars `REPO_URL`, `BRANCH_NAME`, `GITHUB_TOKEN` are provided
- When the container starts
- Then it clones `REPO_URL`, checks out `BRANCH_NAME` (creating the branch if it does not exist), and prints a structured startup message to stdout

**AC #3 — CLAUDE.md is injected before agent launch**
- Given env var `CLAUDE_MD_CONTENT` is set (multi-line string)
- When the entrypoint runs
- Then it writes `CLAUDE_MD_CONTENT` to `/workspace/CLAUDE.md` before invoking Claude Code

**AC #4 — Agent is invoked with PROMPT_CONTENT**
- Given env var `PROMPT_CONTENT` contains the rendered agent prompt
- When the entrypoint runs
- Then Claude Code is invoked with `--print --dangerously-skip-permissions` passing the prompt via stdin or file

**AC #5 — Logs are streamed as NDJSON**
- Given the agent is running
- When it emits output
- Then each line is either a valid JSON object (`{"type":"log","message":"..."}`) or forwarded as-is if already JSON — compatible with `DockerLogStreamer`'s NDJSON parsing

**AC #6 — Cost events are emitted**
- Given the agent session completes
- When Claude Code outputs token usage information
- Then the entrypoint emits a JSON line `{"type":"cost","input_tokens":N,"output_tokens":N,"model":"..."}` to stdout

**AC #7 — Exit code propagation**
- Given the agent exits with code 0 (success) or non-zero (failure)
- When the container exits
- Then the container exit code matches the agent exit code — allowing `AgentRunAction.streamAndWait()` to detect failure

**AC #8 — Non-root execution**
- Given the container is started
- When Claude Code runs
- Then it runs as a non-root user (Claude Code refuses root with `--dangerously-skip-permissions`)

**AC #9 — GitHub auth configured**
- Given `GITHUB_TOKEN` or `CLAUDE_CODE_OAUTH_TOKEN` env vars are set
- When git push or gh CLI commands are executed by the agent
- Then they authenticate successfully without interactive prompts

---

## Tasks / Subtasks

- [ ] **T1.** Create `agent/Dockerfile` — multi-stage build (AC: #1, #2, #8)
  - [ ] T1.1 Base stage: `node:22-bookworm` with git, curl, jq, gh CLI
  - [ ] T1.2 Install Claude Code CLI: `npm install -g @anthropic-ai/claude-code`
  - [ ] T1.3 Create non-root user `agent` with home `/home/agent`, workspace `/workspace`
  - [ ] T1.4 Copy `agent/entrypoint.sh` and make executable
  - [ ] T1.5 Set `ENTRYPOINT ["/entrypoint.sh"]` and `USER agent`

- [ ] **T2.** Create `agent/entrypoint.sh` — runtime entrypoint (AC: #2, #3, #4, #5, #6, #7, #9)
  - [ ] T2.1 Clone `REPO_URL` with token injection for HTTPS auth (same pattern as `scripts/entrypoint.sh` lines 26-29)
  - [ ] T2.2 Check out `BRANCH_NAME` — create if it does not exist on remote, fetch if it does
  - [ ] T2.3 Configure git user identity (`GIT_AUTHOR_NAME`, `GIT_AUTHOR_EMAIL` with defaults)
  - [ ] T2.4 Write `CLAUDE_MD_CONTENT` env var to `/workspace/CLAUDE.md`
  - [ ] T2.5 Write `PROMPT_CONTENT` env var to `/tmp/prompt.md`
  - [ ] T2.6 Configure `gh auth` via token (export `GH_TOKEN=$GITHUB_TOKEN`)
  - [ ] T2.7 Invoke Claude Code: `claude --dangerously-skip-permissions --print < /tmp/prompt.md`
  - [ ] T2.8 Capture exit code and exit with it (propagate to container exit code)
  - [ ] T2.9 Emit cost event line to stdout after agent completes (parse from Claude Code output or emit placeholder)

- [ ] **T3.** Build and tag image locally (AC: #1)
  - [ ] T3.1 Run `docker build -t hopeitworks/agent:latest agent/` from repo root
  - [ ] T3.2 Verify image size is reasonable (< 2GB)

- [ ] **T4.** Smoke test the container (AC: #2, #3, #7)
  - [ ] T4.1 Run container with a real (or test) `REPO_URL`, `BRANCH_NAME`, `CLAUDE_MD_CONTENT`, `PROMPT_CONTENT`
  - [ ] T4.2 Verify it clones, writes CLAUDE.md, invokes Claude, exits with correct code
  - [ ] T4.3 Verify container logs contain NDJSON lines

---

## Dev Notes

### Dependencies

- Docker must be available on the build machine
- `CLAUDE_CODE_OAUTH_TOKEN` required for Claude Code authentication at runtime
- `GITHUB_TOKEN` required for git clone/push of private repos

### File Paths

| File | Purpose |
|------|---------|
| `agent/Dockerfile` | New file — runtime agent image |
| `agent/entrypoint.sh` | New file — container entrypoint |
| `scripts/Dockerfile.dev-agent` | REFERENCE only — do NOT modify |
| `scripts/entrypoint.sh` | REFERENCE only — do NOT modify |
| `backend/internal/adapter/action/agent_run.go` | Consumer of this image — reads exit code and NDJSON logs |

### Technical Specifications

**Dockerfile structure:**

```dockerfile
FROM node:22-bookworm

# System dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl jq \
    && rm -rf /var/lib/apt/lists/*

# GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" \
    | tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
    && apt-get update && apt-get install -y gh && rm -rf /var/lib/apt/lists/*

# Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code

# Non-root user (claude --dangerously-skip-permissions refuses root)
RUN useradd -m -s /bin/bash agent \
    && mkdir -p /workspace /home/agent/.config/gh \
    && chown -R agent:agent /workspace /home/agent

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER agent
WORKDIR /workspace
ENTRYPOINT ["/entrypoint.sh"]
```

**Environment variables consumed by the container:**

| Var | Required | Description |
|-----|----------|-------------|
| `REPO_URL` | Yes | HTTPS URL of the git repository to clone |
| `BRANCH_NAME` | Yes | Branch to checkout (create if absent) |
| `CLAUDE_MD_CONTENT` | Yes | Full content of CLAUDE.md to inject |
| `PROMPT_CONTENT` | Yes | Rendered agent prompt |
| `GITHUB_TOKEN` | Yes | Token for git clone/push and gh CLI auth |
| `CLAUDE_CODE_OAUTH_TOKEN` | Yes | OAuth token for Claude Code |
| `STORY_KEY` | No | Used for git commit context (passed via env) |
| `GIT_AUTHOR_NAME` | No | Git author name (default: `hopeitworks-agent`) |
| `GIT_AUTHOR_EMAIL` | No | Git author email (default: `agent@hopeitworks.local`) |

**NDJSON log format expected by `DockerLogStreamer`:**

Each line from the container that is valid JSON is parsed. The `agent_run.go` consumer reads:
- `{"type":"log","message":"..."}` — forwarded to event system
- `{"type":"cost","input_tokens":N,"output_tokens":N,"model":"..."}` — accumulated for cost recording

Lines that are not valid JSON are wrapped as `{"type":"log","message":"<raw line>"}` by the log streamer.

**Token injection for git clone (from `scripts/entrypoint.sh` reference):**

```bash
CLONE_URL="$REPO_URL"
if [[ -n "${GITHUB_TOKEN:-}" ]] && [[ "$CLONE_URL" == https://github.com/* ]]; then
    CLONE_URL="${CLONE_URL/https:\/\/github.com/https://${GITHUB_TOKEN}@github.com}"
fi
git clone --branch develop --single-branch "$CLONE_URL" /workspace
```

**Branch checkout logic:**

```bash
if git ls-remote --heads origin "$BRANCH_NAME" | grep -q "$BRANCH_NAME"; then
    git fetch origin "$BRANCH_NAME:$BRANCH_NAME"
    git checkout "$BRANCH_NAME"
else
    git checkout -b "$BRANCH_NAME"
fi
```

**Claude Code invocation:**

```bash
echo "$PROMPT_CONTENT" | claude --dangerously-skip-permissions --print
EXIT_CODE=$?
```

The `--print` flag runs Claude Code non-interactively. The prompt is passed via stdin.

**Cost event emission:**

After the agent exits, parse Claude Code's final output for token usage or emit a cost event. Claude Code outputs token usage in JSON when `--output-format json` is used. If not parseable, emit a zero-cost placeholder so the log streamer is not blocked:

```bash
echo '{"type":"cost","input_tokens":0,"output_tokens":0,"model":"unknown"}'
```

### Key Difference vs `scripts/Dockerfile.dev-agent`

The dev-agent image (`scripts/Dockerfile.dev-agent`) is for the BMAD development pipeline — it includes Go tooling, sqlc, oapi-codegen, golangci-lint, and pipeline orchestration scripts. It is heavy (~several GB) and runs the full dev→review→merge cycle.

The **runtime agent image** (`agent/Dockerfile`) is intentionally minimal:
- Only: Node.js + git + gh CLI + Claude Code
- No Go toolchain, no build tools
- Purpose: run a single agent step (implement OR review OR merge) as directed by `PROMPT_CONTENT`
- The prompt template determines what the agent does — the image is generic

### Testing Requirements

- Build the image locally and verify it exits 0
- Run with a mock prompt: `PROMPT_CONTENT="echo hello" REPO_URL=... BRANCH_NAME=test-branch ...`
- Verify `/workspace/CLAUDE.md` exists inside the container after entrypoint executes (use `docker run --entrypoint bash` for inspection)
- Verify non-root user with `docker run ... whoami`

### References

- `scripts/Dockerfile.dev-agent` — reference BMAD dev image (heavier, not for runtime)
- `scripts/entrypoint.sh` — reference BMAD entrypoint (pipeline mode, not for runtime)
- `backend/internal/adapter/action/agent_run.go` — how the image is launched (env vars, container opts)
- `backend/internal/adapter/docker/` — DockerContainerManager and DockerLogStreamer implementations
