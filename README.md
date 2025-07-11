# Book List

### Tooling

The following tools are used in this project:

- **PostgreSQL** – Database engine
- **sqlc** – Generates Go code from SQL queries
- **Air** – Live reloading for Go apps
- **Taskfile** – Task runner for automation
- **Docker Compose** – Multi-container orchestration
- **goose** - Goose is a database migration tool.

### Env Vars Needed

- Create a new .envrc file if you dont already have one
- Add the following info:

```bash
.envrc
export POSTGRES_USER=
export POSTGRES_PASSWORD=
export POSTGRES_DB=
export POSTGRES_HOST=
export POSTGRES_PORT=

export GOOSE_DRIVER=
export GOOSE_DBSTRING="postgres://USER:PASS@HOST:PORT/DB"
export GOOSE_MIGRATION_DIR=./backend/db/migrations
```

- run the following command:
  `direnv allow .`

- you should see the following output:

```bash
direnv: loading ~/devprojects/golang-proj/book-list-app/.envrc
direnv: export +GOOSE_DBSTRING +GOOSE_DRIVER +GOOSE_MIGRATION_DIR +POSTGRES_DB +POSTGRES_HOST +POSTGRES_PASSWORD +POSTGRES_PORT +POSTGRES_USER
```

- then run `env` or `printenv` to see if everything is exported properly
