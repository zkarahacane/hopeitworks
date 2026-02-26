FROM hopeitworks/agent-base:latest

USER root

# Go 1.23
COPY --from=golang:1.23-bookworm /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:/home/agent/go/bin:${PATH}"
ENV GOPATH="/home/agent/go"

# Go dev tools
RUN go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest \
    && go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest \
    && go install github.com/google/wire/cmd/wire@latest \
    && curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b /usr/local/bin

# GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg -o /usr/share/keyrings/githubcli-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

USER agent
