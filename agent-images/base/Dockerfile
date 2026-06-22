# syntax=docker/dockerfile:1
# Builder: runs on the BUILD platform (no QEMU) and cross-compiles agent-runtime to
# the target arch. Pinned by digest (multi-arch manifest list) for reproducibility.
FROM --platform=$BUILDPLATFORM golang:1.23-bookworm@sha256:167053a2bb901972bf2c1611f8f52c44d5fe7e762e5cab213708d82c421614db AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /build
COPY agent-runtime/ ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /agent-runtime .

# Base runtime: pinned by digest (multi-arch manifest list).
FROM debian:bookworm-slim@sha256:96e378d7e6531ac9a15ad505478fcc2e69f371b10f5cdf87857c4b8188404716

# Tool versions (pinned). claude-code AND opencode-ai are both shipped:
# opencode is a supported runtime alongside claude — do not remove either.
ARG CLAUDE_CODE_VERSION=2.1.186
ARG OPENCODE_VERSION=1.17.9

# Core CLI deps shared by every stack: git, gh, ripgrep, jq, curl, ssh.
# gh lives HERE (single source) — stacks must NOT re-install it.
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl jq ca-certificates openssh-client ripgrep gnupg \
    && curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# Node 22 for Claude Code CLI and OpenCode CLI (pinned by digest, multi-arch).
COPY --from=node:22-slim@sha256:d9f850096136edbc402debdd8729579a288aac64574ada0ff4db26b6ae58b0b2 /usr/local/bin/node /usr/local/bin/node
COPY --from=node:22-slim@sha256:d9f850096136edbc402debdd8729579a288aac64574ada0ff4db26b6ae58b0b2 /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

# Install Claude Code CLI and OpenCode CLI (versions pinned, no @latest).
RUN npm install -g \
    @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION} \
    opencode-ai@${OPENCODE_VERSION}

# Agent runtime binary
COPY --from=builder /agent-runtime /usr/local/bin/agent-runtime

# Non-root user
RUN useradd -m -s /bin/bash agent
USER agent
WORKDIR /workspace

ENTRYPOINT ["agent-runtime"]
