# syntax=docker/dockerfile:1
FROM hopeitworks/agent-base:latest

USER root

# Pinned tool versions (no @latest / @HEAD).
ARG SQLC_VERSION=v1.31.1
ARG OAPI_CODEGEN_VERSION=v2.7.1
ARG WIRE_VERSION=v0.7.0
ARG GOLANGCI_LINT_VERSION=v2.12.2
ARG GOPLS_VERSION=v0.22.0
ARG GOVULNCHECK_VERSION=v1.4.0

# Go 1.23 (toolchain copied from pinned-by-digest golang image, multi-arch).
COPY --from=golang:1.23-bookworm@sha256:167053a2bb901972bf2c1611f8f52c44d5fe7e762e5cab213708d82c421614db /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:/home/agent/go/bin:${PATH}"
ENV GOPATH="/home/agent/go"

# Go dev tools (pinned) + LSP (gopls) + vuln scanner (govulncheck).
# gh comes from agent-base — do not re-install it here.
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION} \
    && go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION} \
    && go install github.com/google/wire/cmd/wire@${WIRE_VERSION} \
    && go install golang.org/x/tools/gopls@${GOPLS_VERSION} \
    && go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/${GOLANGCI_LINT_VERSION}/install.sh | sh -s -- -b /usr/local/bin ${GOLANGCI_LINT_VERSION}

USER agent
