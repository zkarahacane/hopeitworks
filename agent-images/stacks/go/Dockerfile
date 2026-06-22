# syntax=docker/dockerfile:1

# Pinned tool versions (no @latest / @HEAD).
ARG SQLC_VERSION=v1.31.1
ARG OAPI_CODEGEN_VERSION=v2.7.1
ARG WIRE_VERSION=v0.7.0
ARG GOLANGCI_LINT_VERSION=v1.64.8
ARG GOPLS_VERSION=v0.22.0
ARG GOVULNCHECK_VERSION=v1.4.0

# Go dev tools cross-compiled on the BUILD platform (no QEMU): GOARCH=$TARGETARCH.
# Compiling gopls/sqlc under arm64 emulation took 25+ min; cross-compiling from the
# native builder is a few minutes. Pinned by digest (multi-arch manifest list).
FROM --platform=$BUILDPLATFORM golang:1.23-bookworm@sha256:167053a2bb901972bf2c1611f8f52c44d5fe7e762e5cab213708d82c421614db AS gotools
ARG TARGETOS
ARG TARGETARCH
ARG SQLC_VERSION
ARG OAPI_CODEGEN_VERSION
ARG WIRE_VERSION
ARG GOPLS_VERSION
ARG GOVULNCHECK_VERSION
# GOTOOLCHAIN=auto: the golang image pins it to "local", but sqlc@v1.31.1 needs
# go >= 1.26; auto lets `go install` fetch the required toolchain (host-native).
ENV GOPATH=/go GOTOOLCHAIN=auto
# go install refuses GOBIN for cross-compiled binaries and drops them under
# $GOPATH/bin/<os>_<arch>/ when cross-compiling ($GOPATH/bin when native).
# Normalise both layouts into /gotools for a single COPY in the final stage.
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION} \
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@${OAPI_CODEGEN_VERSION} \
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install github.com/google/wire/cmd/wire@${WIRE_VERSION} \
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install golang.org/x/tools/gopls@${GOPLS_VERSION} \
    && GOOS=${TARGETOS} GOARCH=${TARGETARCH} go install golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION} \
    && mkdir -p /gotools \
    && if [ -d "/go/bin/${TARGETOS}_${TARGETARCH}" ]; then cp /go/bin/${TARGETOS}_${TARGETARCH}/* /gotools/; else cp /go/bin/* /gotools/; fi

FROM hopeitworks/agent-base:latest

USER root

ARG GOLANGCI_LINT_VERSION

# Go 1.23 (toolchain copied from pinned-by-digest golang image, multi-arch).
COPY --from=golang:1.23-bookworm@sha256:167053a2bb901972bf2c1611f8f52c44d5fe7e762e5cab213708d82c421614db /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:/home/agent/go/bin:${PATH}"
ENV GOPATH="/home/agent/go"

# Cross-compiled Go dev tools + LSP (gopls) + vuln scanner (govulncheck), from the
# gotools stage. golangci-lint ships a prebuilt binary (install.sh only downloads).
# gh comes from agent-base — do not re-install it here.
COPY --from=gotools /gotools/ /home/agent/go/bin/
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/${GOLANGCI_LINT_VERSION}/install.sh | sh -s -- -b /usr/local/bin ${GOLANGCI_LINT_VERSION}

USER agent
