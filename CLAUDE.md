# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All automation runs through [Taskfile](https://taskfile.dev). Run from the repo root:

| Command | Description |
|---|---|
| `task init` | First-time setup: creates `.env`, starts DB, runs migrations, generates sqlc code |
| `task dev` | Start the backend with live reload via Air (also runs gen + migrate) |
| `task db:up` / `task db:down` | Start / stop PostgreSQL via Docker Compose |
| `task db:migrate` | Apply pending goose migrations |
| `task gen` | Regenerate `pkg/database/` from SQL using sqlc |
| `task doctor` | Verify toolchain and DB connectivity |

The app runs on port **8080**. PostgreSQL defaults: user/password/db all `app`, port `5432`.

There is no test suite yet.

## Architecture

This is a Go HTTP backend (stdlib `net/http`, no framework) that wraps the [Open Library API](https://openlibrary.org/) and stores a personal reading list in PostgreSQL.

**Request flow:**
```
HTTP request → handler (backend/handlers/) → sqlc Queries (backend/pkg/database/) → PostgreSQL
                                           ↘ Open Library API (external HTTP)
```

**Key design points:**

- **sqlc** (`backend/sqlc.yaml`) generates type-safe Go from `.sql` files. Source of truth for DB access is `backend/db/queries/*.sql`; never edit `backend/pkg/database/*.go` by hand — regenerate with `task gen`.
- **goose** manages migrations in `backend/db/migrations/`. New migration files must follow the `YYYYMMDDHHMMSS_description.sql` naming convention and use `-- +goose Up` / `-- +goose Down` annotations.
- **`BookHandler`** (`handlers/book_handler.go`) — stateless, proxies search and detail lookups to Open Library.
- **`ReadlistHandler`** (`handlers/readlist_handler.go`) — holds `*sql.DB` and `*database.Queries`; manages the local reading list.
- **`handlers/helpers.go`** — shared `WriteJSON` / `WriteError` used by all handlers. All error responses are `{"error": "message"}` JSON.
- The `books` table stores `work_id` (from Open Library's `/works/{id}` path) as a unique key. `authors` and `subjects` are stored as comma-separated strings. Adding a duplicate `work_id` returns 409.
- The router (`chi`) lives in `cmd/main.go`. Method enforcement, middleware (request ID, logging, panic recovery), and graceful shutdown are all wired there.

**Routes:**
| Method | Path | Auth | Handler |
|---|---|---|---|
| `GET` | `/health` | public | DB ping check |
| `GET` | `/search?q=` | public | Open Library search proxy |
| `GET` | `/details?id=` | public | Open Library work details proxy |
| `GET` | `/readlist` | required | List caller's saved books |
| `POST` | `/readlist` | required | Add a book (body: title, authors, work_id, subjects?, description?, cover_art_url?) |
| `GET` | `/readlist/{workID}` | required | Get a single book by Open Library work ID |
| `PATCH` | `/readlist/{id}` | required | Update status, rating (1–5), or notes (all optional, unset fields preserved) |
| `DELETE` | `/readlist/{id}` | required | Remove a book by internal numeric ID |

**`books` table schema** (key fields):
- `user_id TEXT NOT NULL` — Keycloak `sub`; all queries are scoped to this
- `status TEXT NOT NULL DEFAULT 'want_to_read'` — one of: `want_to_read`, `reading`, `finished`, `abandoned`
- `rating INTEGER` — nullable, 1–5
- `notes TEXT` — nullable
- `UNIQUE(work_id, user_id)` — same Open Library book can appear in multiple users' lists

## Auth (Keycloak)

Auth is implemented as a chi middleware in `backend/auth/middleware.go`. All `/readlist/*` routes are protected; `/health`, `/search`, `/details` are public.

**How it works:** The middleware validates the `Authorization: Bearer <token>` header using Keycloak's JWKS endpoint (via `go-oidc`). On success, the Keycloak `sub` (UUID) is available via `auth.SubFromContext(r.Context())` — this is the user identity for Phase 3.

**Two env vars control Keycloak connectivity** (see `.envrc.example`):
- `KEYCLOAK_ISSUER` — the `iss` claim in tokens; the public-facing URL clients use (`http://localhost:8180/realms/booklist`)
- `KEYCLOAK_DISCOVERY_URL` — where the server fetches OIDC config; differs inside the devcontainer (`http://keycloak:8080/realms/booklist`). The devcontainer.json sets this automatically.

**One-time Keycloak setup** (after `task init`):
1. Open `http://localhost:8180`, log in as `admin/admin`
2. Create realm: `booklist`
3. Create client: `booklist-api`, type OpenID Connect, authentication OFF, Direct Access Grants ON
4. Create a test user with a password
5. In Bruno, run `GET token` to get a bearer token, then use it on readlist requests

## Environment

Environment is managed via `.env` (loaded by Taskfile) or `.envrc` + direnv. Copy `.envrc.example` to `.envrc` and run `direnv allow .`. The required variables are `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_PORT`, `POSTGRES_HOST`, and derived `DB_URL`.

Inside the dev container, `POSTGRES_HOST` must be `host.docker.internal` to reach the Dockerized DB from within the container. Keycloak env vars are set automatically via `devcontainer.json` `containerEnv`.

## API Testing

Bruno collections are in `utils/Bruno/book-api/`. Open this directory in [Bruno](https://www.usebruno.com/) to test endpoints locally.
