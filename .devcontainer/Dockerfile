FROM mcr.microsoft.com/devcontainers/go:1-1.22-bookworm

# OS packages
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    postgresql-client ca-certificates curl unzip make git direnv \
 && rm -rf /var/lib/apt/lists/*

# Ensure Go tool + Go bin are on PATH for all shells
ENV PATH="/usr/local/go/bin:/usr/local/bin:${PATH}"
ENV GOBIN="/usr/local/bin"

# Pin tool versions and install as root so binaries land in /usr/local/bin
ARG TASK_VERSION=v3.39.2
ARG AIR_VERSION=v1.52.3
ARG SQLC_VERSION=v1.27.0
ARG GOOSE_VERSION=v3.20.0

RUN echo 'eval "$(direnv hook bash)"' >> /etc/bash.bashrc && \
    echo 'eval "$(direnv hook zsh)"'  >> /etc/zsh/zshrc

RUN go version && \
    go install github.com/go-task/task/v3/cmd/task@${TASK_VERSION} && \
    go install github.com/air-verse/air@${AIR_VERSION} && \   
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION} && \
    go install github.com/pressly/goose/v3/cmd/goose@${GOOSE_VERSION} && \
    task --version && air -v && sqlc version && goose -version
