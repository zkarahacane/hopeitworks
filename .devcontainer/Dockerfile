FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

ARG TARGETARCH
ARG GO_VERSION=1.24.4
ARG NODE_VERSION=22
ARG GOLANGCI_LINT_VERSION=latest
ARG SQLC_VERSION=v1.28.0
ARG MIGRATE_VERSION=v4.18.3

# ─── System deps ───────────────────────────────────────────────
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential \
    curl \
    wget \
    git \
    jq \
    unzip \
    ca-certificates \
    gnupg \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# ─── Go ────────────────────────────────────────────────────────
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" \
    | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:/home/vscode/go/bin:${PATH}"
ENV GOPATH="/home/vscode/go"

# ─── Node (via NodeSource) ─────────────────────────────────────
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_VERSION}.x | bash - \
    && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# ─── pnpm ──────────────────────────────────────────────────────
RUN corepack enable && corepack prepare pnpm@latest --activate

# ─── Docker CLI (socket mounted from host) ─────────────────────
RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc \
    && chmod a+r /etc/apt/keyrings/docker.asc \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] \
       https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
       > /etc/apt/sources.list.d/docker.list \
    && apt-get update && apt-get install -y docker-ce-cli docker-compose-plugin \
    && rm -rf /var/lib/apt/lists/*

# ─── GitHub CLI ────────────────────────────────────────────────
RUN (type -p wget >/dev/null || apt-get install wget -y) \
    && mkdir -p -m 755 /etc/apt/keyrings \
    && out=$(mktemp) && wget -nv -O$out https://cli.github.com/packages/githubcli-archive-keyring.gpg \
    && cat $out | tee /etc/apt/keyrings/githubcli-archive-keyring.gpg > /dev/null \
    && chmod go+r /etc/apt/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/githubcli-archive-keyring.gpg] \
       https://cli.github.com/packages stable main" \
       > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update && apt-get install gh -y \
    && rm -rf /var/lib/apt/lists/*

# ─── Go tools (as vscode user for correct GOPATH) ──────────────
USER vscode

RUN go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest \
    && go install github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION} \
    && go install github.com/google/wire/cmd/wire@latest

USER root

# ─── golangci-lint ─────────────────────────────────────────────
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
    | sh -s -- -b /usr/local/bin ${GOLANGCI_LINT_VERSION}

# ─── golang-migrate ────────────────────────────────────────────
RUN curl -fsSL "https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-${TARGETARCH}.tar.gz" \
    | tar -C /usr/local/bin -xz migrate

# ─── Claude Code CLI ──────────────────────────────────────────
RUN npm install -g @anthropic-ai/claude-code

# ─── Playwright system deps (for e2e tests) ───────────────────
RUN npx playwright install-deps chromium

# ─── Docker group for socket access ───────────────────────────
RUN groupadd -f docker && usermod -aG docker vscode

USER vscode
