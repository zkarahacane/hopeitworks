# hopeitworks Runtime Agent Container
# Minimal image for running a single Claude Code agent step.
# Receives: REPO_URL, BRANCH_NAME, CLAUDE_MD_CONTENT, PROMPT_CONTENT
# Produces: NDJSON logs on stdout, exit code propagated from agent
#
# Key difference from scripts/Dockerfile.dev-agent:
#   - No Go toolchain, no build tools, no pipeline orchestration
#   - Only: Node.js + git + gh CLI + Claude Code
#   - Purpose: run one agent step as directed by PROMPT_CONTENT

FROM node:22-bookworm

# System dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    curl \
    jq \
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
