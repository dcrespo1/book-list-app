# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

All automation runs through [Taskfile](https://taskfile.dev). Run from the repo root:

| Command | Description |
|---|---|
| `task init` | First-time setup: creates `.env`, starts DB + Keycloak, runs migrations |
| `task dev` | Start the backend with live reload via Air (also runs migrate) |
| `task db:up` / `task db:down` | Start / stop PostgreSQL via Docker Compose |
| `task kc:up` / `task kc:wait` | Start Keycloak / wait until ready |
| `task db:migrate` | Apply pending goose migrations |
| `task gen` | Regenerate `pkg/database/` from SQL using sqlc |
| `task test` | Run `go test ./...` |
| `task lint` | Run `go vet ./...` |
| `task doctor` | Verify toolchain and DB connectivity |

Backend runs on port **8080**. Frontend dev server on port **5173**. PostgreSQL defaults: user/password/db all `app`, port `5432`. Keycloak on port **8180**.

## Repository Layout

```
backend/    Go HTTP API
frontend/   SvelteKit SPA
utils/      Docker Compose, Bruno collections, Keycloak setup guide
```

---

## Backend

### Architecture

Go HTTP backend (stdlib `net/http` + chi router) that wraps the [Open Library API](https://openlibrary.org/) and stores a personal reading list in PostgreSQL.

**Request flow:**
```
HTTP request → chi middleware (CORS, auth) → handler (backend/handlers/)
                                           → sqlc Queries (backend/pkg/database/) → PostgreSQL
                                           → Open Library API (external HTTP)
```

**Key design points:**

- **sqlc** (`backend/sqlc.yaml`) generates type-safe Go from `.sql` files. Source of truth is `backend/db/queries/*.sql`; never edit `backend/pkg/database/*.go` by hand — regenerate with `task gen`.
- **goose** manages migrations in `backend/db/migrations/`. New files must follow `YYYYMMDDHHMMSS_description.sql` and use `-- +goose Up` / `-- +goose Down` annotations.
- **`BookHandler`** (`handlers/book_handler.go`) — stateless, proxies search and detail lookups to Open Library.
- **`ReadlistHandler`** (`handlers/readlist_handler.go`) — holds `*database.Queries` via the `BookStore` interface; manages the local reading list.
- **`BookStore` interface** (`handlers/readlist_handler.go`) — abstracts DB access; `*database.Queries` satisfies it in production, `fakeStore` in tests.
- **`BookResponse` DTO** (`handlers/response.go`) — all readlist endpoints return this type, not `database.Book` directly. Fields are snake_case, nullable fields are Go pointers (`*string`, `*int32`), `user_id` is excluded.
- **`handlers/helpers.go`** — shared `WriteJSON` / `WriteError`. All error responses are `{"error": "message"}` JSON.
- **CORS** (`go-chi/cors`) — wired in `cmd/main.go` before other middleware; allows `http://localhost:5173`.
- The `books` table stores `work_id` (Open Library `/works/{id}` path) as a unique key per user. `authors` and `subjects` are comma-separated strings. Duplicate `work_id` per user returns 409.
- The router (chi) lives in `cmd/main.go` with middleware: CORS, request ID, logger, panic recovery, graceful shutdown.

### Routes

| Method | Path | Auth | Handler |
|---|---|---|---|
| `GET` | `/health` | public | DB ping check |
| `GET` | `/search?q=` | public | Open Library search proxy |
| `GET` | `/details?id=` | public | Open Library work details proxy |
| `GET` | `/readlist/` | required | List caller's saved books |
| `POST` | `/readlist/` | required | Add a book (body: title, authors, work_id, subjects?, description?, cover_art_url?) |
| `GET` | `/readlist/{workID}` | required | Get a single book by Open Library work ID |
| `PATCH` | `/readlist/{id}` | required | Update status, rating (1–5), or notes (all optional, unset fields preserved) |
| `DELETE` | `/readlist/{id}` | required | Remove a book by internal numeric ID; returns 404 if not found |

### `books` table schema (key fields)

- `user_id TEXT NOT NULL` — Keycloak `sub`; all queries are scoped to this
- `status TEXT NOT NULL DEFAULT 'want_to_read'` — one of: `want_to_read`, `reading`, `finished`, `abandoned`
- `rating INTEGER` — nullable, 1–5
- `notes TEXT` — nullable
- `UNIQUE(work_id, user_id)` — same Open Library book can appear in multiple users' lists

### Tests

Tests exist for all handlers and auth middleware. Run with `task test`.

- `backend/auth/middleware_test.go` — auth middleware unit tests using `mockVerifier`
- `backend/handlers/readlist_handler_test.go` — handler tests using `fakeStore` (no real DB)
- `backend/handlers/book_handler_test.go` — Open Library proxy tests using a local `httptest.Server`

---

## Auth (Keycloak)

Auth is implemented as chi middleware in `backend/auth/middleware.go`. All `/readlist/*` routes are protected; `/health`, `/search`, `/details` are public.

**How it works:** The middleware validates `Authorization: Bearer <token>` using Keycloak's JWKS endpoint (via `go-oidc`). On success the Keycloak `sub` (UUID) is injected into context via `auth.SubFromContext(r.Context())`.

**`TokenVerifier` interface** (`backend/auth/middleware.go`) — wraps `*oidc.IDTokenVerifier`; `mockVerifier` satisfies it in tests.

**Two env vars control Keycloak connectivity:**
- `KEYCLOAK_ISSUER` — the `iss` claim in tokens; public-facing URL (`http://localhost:8180/realms/booklist`)
- `KEYCLOAK_DISCOVERY_URL` — where the server fetches OIDC config; differs inside devcontainer (`http://keycloak:8080/realms/booklist`)

**One-time Keycloak setup:** see `utils/keycloak-setup.md` for the full step-by-step guide including Keycloak 26 gotchas (required user profile fields, password reset flow).

---

## Frontend

SvelteKit SPA (CSR only, SSR disabled) in `frontend/`.

### Stack

- **SvelteKit** with Svelte 5 runes mode
- **Tailwind CSS v4** for styling
- **Skeleton v3** (`@skeletonlabs/skeleton` + `@skeletonlabs/skeleton-svelte`) — UI components, cerberus theme
- **keycloak-js** — PKCE redirect flow, silent SSO, token refresh
- **`$env/static/public`** — Vite env vars prefixed `PUBLIC_`

### Running the frontend

```bash
cd frontend
npm install
npm run dev       # dev server at http://localhost:5173
npm run build     # production build
```

Copy `.env.example` to `.env` — defaults point to `localhost:8080` (backend) and `localhost:8180` (Keycloak).

### Structure

```
frontend/src/
  lib/
    auth.ts          keycloak instance, isAuthenticated/isInitialized stores, login/logout
    api.ts           fetch wrapper (auto-attaches Bearer token, refreshes before expiry)
                     BookResponse and SearchResult TypeScript types
  routes/
    +layout.ts       export const ssr = false  (global CSR)
    +layout.svelte   initialises Keycloak on mount; shows loading until ready
    +page.svelte     landing page — redirects to /readlist if already signed in
    (protected)/
      +layout.svelte auth guard (redirects to Keycloak login if not authenticated) + nav bar
      search/
        +page.svelte search Open Library; per-card add state (idle/adding/saved/duplicate/error)
      readlist/
        +page.svelte reading list; inline status select, star rating, click-to-edit notes, delete
```

### Key patterns

- **Auth init:** `initAuth()` called in root layout `onMount`; uses `check-sso` + PKCE S256 + silent SSO iframe
- **API calls:** always go through `api.ts` which calls `keycloak.updateToken(30)` before each request
- **Per-card state:** search results track add status per `work_id` in a reactive record
- **Inline editing:** readlist notes use click-to-edit; PATCH fires on textarea blur only if content changed
- **Optimistic updates:** status and rating changes update local state immediately after a successful PATCH response

### Static files

- `frontend/static/silent-check-sso.html` — required by keycloak-js silent SSO check

---

## Environment

Backend environment is managed via `.env` (loaded by Taskfile). Required vars: `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `POSTGRES_PORT`, `POSTGRES_HOST`, and derived `DB_URL`.

Frontend environment uses `frontend/.env` with `PUBLIC_` prefixed vars: `PUBLIC_KEYCLOAK_URL`, `PUBLIC_KEYCLOAK_REALM`, `PUBLIC_KEYCLOAK_CLIENT_ID`, `PUBLIC_API_URL`.

---

## API Testing

Bruno collections are in `utils/Bruno/book-api/`. Run **GET token** first to get a bearer token, then use it on readlist requests. See `utils/keycloak-setup.md` for credentials.
