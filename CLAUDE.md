# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Greenlight is a RESTful JSON API for managing and discovering movies. Built in Go with Chi router, PostgreSQL (pgx/v5), sqlc for type-safe query generation, and Goose for migrations.

## Commands

```bash
# Run the API
make run/locally          # Connect to localhost PostgreSQL
make run/homelab          # Connect to homelab PostgreSQL (192.168.1.60)

# Database
make db/up                # Apply all pending Goose migrations
make db/migrations/new name=<name>  # Create a new migration file

# Quality & Testing
make audit                # fmt, vet, staticcheck, go test -race ./...
make tidy                 # go mod tidy + go fmt + go mod verify

# Build
make build/api            # Produces binary + linux/amd64 cross-compile
```

Run a single test:
```bash
go test ./internal/movies/... -run TestRuntimeUnmarshalJSON
go test ./cmd/api/... -run TestMovieHandler
```

## Architecture

### Request Flow

```
HTTP Request
  → RecordRequestMetrics → Logger → CORS → Recoverer → RateLimit → Authenticate
  → (protected routes) RequireActivatedUser → RequirePermission
  → Handler → Service → sqlc Querier → PostgreSQL
```

### Package Layout

**`cmd/api/`** — Application wiring and HTTP layer:
- `main.go` — Config parsing (all via CLI flags), dependency injection, service initialization
- `routes.go` — Chi router and middleware stack setup
- `middleware.go` — `Authenticate`, `RequireActivatedUser`, `RequirePermission`
- `handlers/` pattern: `movies.go`, `users.go`, `tokens.go` — thin handlers that parse input, validate, call services, write JSON
- `helpers.go` — `readJSON`, `writeJSON`, query string parsing
- `errors.go` — Standardized HTTP error responses
- `metrics.go` — Expvar-based metrics; accessible at `GET /debug/vars`
- `context.go` — Auth user stored/retrieved via request context

**`internal/`** — Domain logic:
- `movies/` — Movie model, custom `Runtime` JSON marshaling (`"120 mins"`), validation, `MovieService` over sqlc querier
- `users/` — User model (bcrypt cost 12), `UserService`, token generation and authentication
- `permissions/` — `PermissionService` for `movies:read` / `movies:write` checks
- `mailer/` — SMTP sender with 3-attempt retry; templates via `go:embed`
- `validator/` — Accumulating error validator used across handlers
- `data/sqlc/` — **Generated** Go code from `queries.sql` — do not hand-edit
- `data/migrations/` — Goose SQL migration files

### Key Patterns

**Configuration:** Everything via CLI flags — no config files. `.env` provides DB DSNs only for `make` targets (Goose + app flag).

**Database access:** sqlc generates a `Querier` interface from `internal/data/sqlc/queries.sql`. Services accept `repo.Querier`; run `sqlc generate` after editing SQL queries.

**Optimistic locking:** Movies use a `version` integer; `UPDATE ... WHERE id = $1 AND version = $2` returns `ErrEditConflict` on mismatch.

**Authentication:** Bearer tokens — plaintext returned to client, SHA-256 hash stored in DB. Scopes: `activation` (3-day TTL) and `authentication` (24-hour TTL).

**Background tasks:** Use `app.background(func() {...})` which wraps goroutines with panic recovery and tracks them in a `sync.WaitGroup` for graceful shutdown.

**Error types:** `data.ErrRecordNotFound`, `users.ErrDuplicateEmail`, `data.ErrEditConflict` — handlers map these to appropriate HTTP status codes via `errors.go`.

## API Routes

| Method | Path | Permission |
|--------|------|------------|
| GET | `/v1/healthcheck` | Public |
| POST | `/v1/users` | Public |
| PUT | `/v1/users/activated` | Public |
| POST | `/v1/tokens/authentication` | Public |
| GET | `/debug/vars` | Public |
| GET | `/v1/movies` | `movies:read` |
| POST | `/v1/movies` | `movies:write` |
| GET | `/v1/movies/{id}` | `movies:read` |
| PATCH | `/v1/movies/{id}` | `movies:write` |
| DELETE | `/v1/movies/{id}` | `movies:write` |

## Database

Migrations live in `internal/data/migrations/` and are managed with Goose. The `.env` file at the project root sets `GREENLIGHT_DB_DSN` and `GREENLIGHT_HOMELAB_DB_DSN`.

After editing `internal/data/sqlc/queries.sql` or migration schemas, regenerate with:
```bash
sqlc generate
```
