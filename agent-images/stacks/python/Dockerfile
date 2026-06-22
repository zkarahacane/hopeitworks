# syntax=docker/dockerfile:1
FROM hopeitworks/agent-base:latest

USER root

# Pinned Python tooling versions.
ARG UV_VERSION=0.11.23
ARG RUFF_VERSION=0.15.18
ARG MYPY_VERSION=2.1.0
ARG PYTEST_VERSION=9.1.1
ARG PYRIGHT_VERSION=1.1.410

# Python 3.12 (Debian bookworm ships 3.11 as python3; install 3.12 explicitly).
# gh comes from agent-base — do not re-install it here.
RUN apt-get update && apt-get install -y --no-install-recommends \
    python3.12 python3.12-venv python3-pip pipx \
    && rm -rf /var/lib/apt/lists/* \
    && ln -sf /usr/bin/python3.12 /usr/bin/python3 \
    && ln -sf /usr/bin/python3.12 /usr/bin/python

# Python dev tools (pinned) + LSP (pyright). pipx isolates each tool;
# uv, ruff, mypy, pytest, pyright land on PATH at /usr/local/bin.
ENV PIPX_HOME=/opt/pipx
ENV PIPX_BIN_DIR=/usr/local/bin
RUN pipx install uv==${UV_VERSION} \
    && pipx install ruff==${RUFF_VERSION} \
    && pipx install mypy==${MYPY_VERSION} \
    && pipx install pytest==${PYTEST_VERSION} \
    && pipx install pyright==${PYRIGHT_VERSION}

USER agent
