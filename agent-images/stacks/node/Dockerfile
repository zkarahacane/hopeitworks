# syntax=docker/dockerfile:1
FROM hopeitworks/agent-base:latest

USER root

# Pinned tool versions (no @latest). Node + npm come from agent-base.
ARG TYPESCRIPT_VERSION=6.0.3
ARG PRETTIER_VERSION=3.8.4
ARG ESLINT_VERSION=10.5.0
ARG TS_LANGUAGE_SERVER_VERSION=5.3.0
ARG PNPM_VERSION=11.8.0

# Node dev tools (pinned) + LSP (typescript-language-server) + pnpm.
# gh comes from agent-base — do not re-install it here.
RUN npm install -g \
    typescript@${TYPESCRIPT_VERSION} \
    prettier@${PRETTIER_VERSION} \
    eslint@${ESLINT_VERSION} \
    typescript-language-server@${TS_LANGUAGE_SERVER_VERSION} \
    pnpm@${PNPM_VERSION}

USER agent
