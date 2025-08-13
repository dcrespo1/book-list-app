# Book List App
A Go-based backend application with PostgreSQL, built for rapid local development using Docker Compose, Taskfile, Air (live reload), sqlc (type-safe SQL), and goose (migrations). The project includes a VS Code Dev Container for a fully reproducible, cross-platform dev environment.

## Features

- Go Backend with hot reload via Air

- PostgreSQL for persistent data storage

- Database migrations using Goose

- Type-safe DB access with sqlc

- Taskfile-driven automation for a consistent DX

- Dev Container setup for zero-install onboarding

- direnv integration for environment variable management

- Docker Compose for local infrastructure

- Bruno collection for API testing

## Prerequisites

- Docker Desktop (with WSL integration if on Windows)

- VS Code

- VS Code Dev Containers extension

- direnv (optional, for local env management)

## Getting Started
### 1. Clone the repository
```bash
git clone git@github.com:dcrespo1/book-list-app.git

cd book-list-app
```
### 2. Review and adjust per-user settings (important)

Before opening in the dev container, check the following:

- SSH key type & mounts
  - In `.devcontainer/devcontainer.json`, update the "mounts" paths to point to your actual SSH key files.

  - If you use ed25519 keys (common):
  ```bash
  "source=${env:HOME}/.ssh/id_ed25519,target=/home/vscode/.ssh/id_ed25519,type=bind,consistency=cached",
  "source=${env:HOME}/.ssh/id_ed25519.pub,target=/home/vscode/.ssh/id_ed25519.pub,type=bind,consistency=cached"
  ```

  - If you use rsa keys or a custom filename, change these accordingly.

  - Ensure your SSH key is added to your GitHub account.

### 3. Open in VS Code and Reopen in Container
1. Open the repo in VS Code

2. When prompted, choose "Reopen in Container"
The Dev Container will:

   - Build the image with all tools installed

   - Mount your SSH keys

   - Set up PostgreSQL

   - Run task init to generate code and run migrations

## Development Workflow
### Common tasks are automated via Taskfile.
| Command           | Description                                                              |
| ----------------- | ------------------------------------------------------------------------ |
| `task init`       | First-time setup: seed `.envrc`, start DB, run migrations, generate code |
| `task db:up`      | Start PostgreSQL via Docker Compose                                      |
| `task db:down`    | Stop PostgreSQL                                                          |
| `task db:logs`    | Tail PostgreSQL logs                                                     |
| `task db:migrate` | Apply DB migrations                                                      |
| `task gen`        | Generate Go code from SQL (`sqlc`)                                       |
| `task dev`        | Run app with live reload (Air)                                           |
| `task doctor`     | Check toolchain and DB connectivity                                      |

## Directory Structure
```bash
.
├── backend/                  # Go backend source code
│   ├── cmd/                   # Application entrypoints
│   ├── db/                    # SQL queries, migrations
│   └── ...
├── taskfiles/                 # Additional task definitions
├── utils/
│   ├── docker-compose.yaml    # Local infra definitions
│   └── Bruno/                 # API testing collections
├── .devcontainer/             # Dev Container configuration
├── Taskfile.yaml              # Main automation entrypoint
└── README.md

```

## Env Vars Needed

- Create a new .envrc file if you dont already have one
- Add the following info:

```bash
# .envrc.example
# Load .env first if present
dotenv_if_exists .env

# Defaults (only if not set)
export POSTGRES_USER=${POSTGRES_USER:=app}
export POSTGRES_PASSWORD=${POSTGRES_PASSWORD:=app}
export POSTGRES_DB=${POSTGRES_DB:=app}
export POSTGRES_PORT=${POSTGRES_PORT:=5432}
export POSTGRES_HOST=${POSTGRES_HOST:=host.docker.internal}

# Derivatives
export DB_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"
export PORT=${PORT:=8080}
export ENV=${ENV:=development}
```

- run the following command:
  `direnv allow .`

- you should see the following output:

```bash
direnv: loading ~/devprojects/golang-proj/book-list-app/.envrc
direnv: export +GOOSE_DBSTRING +GOOSE_DRIVER +GOOSE_MIGRATION_DIR +POSTGRES_DB +POSTGRES_HOST +POSTGRES_PASSWORD +POSTGRES_PORT +POSTGRES_USER
```

- then run `env` or `printenv` to see if everything is exported properly

## API Testing
- The repo includes a Bruno collection under `utils/Bruno`.
- Open this in Bruno to test endpoints locally.

## Troubleshooting
- Postgres connection refused: Ensure DB is running with task db:up and that POSTGRES_HOST is host.docker.internal inside the dev container.

- SSH key issues: Confirm your key is mounted via mounts in .devcontainer/devcontainer.json and that permissions are correct (chmod 600 for private keys).

- Email privacy block on push: See GitHub GH007 docs to use your noreply email or disable the check.