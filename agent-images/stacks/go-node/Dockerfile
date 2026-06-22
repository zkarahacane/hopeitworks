# syntax=docker/dockerfile:1
# go-node = real union of the go and node stacks (NOT a copy of go).
FROM hopeitworks/agent-base:latest

USER root

# Pinned Go tool versions.
ARG SQLC_VERSION=v1.31.1
ARG OAPI_CODEGEN_VERSION=v2.7.1
ARG WIRE_VERSION=v0.7.0
ARG GOLANGCI_LINT_VERSION=v2.12.2
ARG GOPLS_VERSION=v0.22.0
ARG GOVULNCHECK_VERSION=v1.4.0
# Pinned Node tool versions.
ARG TYPESCRIPT_VERSION=6.0.3
ARG PRETTIER_VERSION=3.8.4
ARG ESLINT_VERSION=10.5.0
ARG TS_LANGUAGE_SERVER_VERSION=5.3.0
ARG PNPM_VERSION=11.8.0

# --- Go side ---
# Go 1.23 (toolchain copied from pinned-by-digest golang image, multi-arch).
COPY --from=golang:1.23-bookworm@sha256:167053a2bb901972bf2c1611f8f52c44d5fe7e762e5cab213708d82c421614db /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:/home/agent/go/bin:${PATH}"
ENV GOPATH="/home/agent/go"

# Go dev tools (pinned) + LSP (gopls) + vuln scanner (govulncheck).
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION} \
    && go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION} \
    && go install github.com/google/wire/cmd/wire@${WIRE_VERSION} \
    && go install golang.org/x/tools/gopls@${GOPLS_VERSION} \
    && go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/${GOLANGCI_LINT_VERSION}/install.sh | sh -s -- -b /usr/local/bin ${GOLANGCI_LINT_VERSION}

# --- Node side (Node + npm already in agent-base) ---
# Node dev tools (pinned) + LSP (typescript-language-server) + pnpm.
RUN npm install -g \
    typescript@${TYPESCRIPT_VERSION} \
    prettier@${PRETTIER_VERSION} \
    eslint@${ESLINT_VERSION} \
    typescript-language-server@${TS_LANGUAGE_SERVER_VERSION} \
    pnpm@${PNPM_VERSION}

# gh comes from agent-base — do not re-install it here.

USER agent
