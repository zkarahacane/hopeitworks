FROM golang:1.23-bookworm AS builder
WORKDIR /build
COPY agent-runtime/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /agent-runtime .

FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl jq ca-certificates openssh-client \
    && rm -rf /var/lib/apt/lists/*

# Node 22 for Claude Code CLI and OpenCode CLI
COPY --from=node:22-slim /usr/local/bin/node /usr/local/bin/node
COPY --from=node:22-slim /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s /usr/local/lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

# Install Claude Code CLI and OpenCode CLI
RUN npm install -g @anthropic-ai/claude-code opencode-ai

# Agent runtime binary
COPY --from=builder /agent-runtime /usr/local/bin/agent-runtime

# Non-root user
RUN useradd -m -s /bin/bash agent
USER agent
WORKDIR /workspace

ENTRYPOINT ["agent-runtime"]
