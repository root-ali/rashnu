# Rashnu — agent guide

Instructions for AI agents (Cursor, Claude Code, etc.) working in this repository.

## Project

**Rashnu** is a FinOps platform for bare-metal infrastructure: track datacenters, servers, network hardware, service workloads, depreciation pricing, and per-service cost reports.

- **Backend**: Go 1.26, chi router, pgx, zap, JWT auth — `cmd/server`, `internal/*`
- **Frontend**: React 18, TypeScript, Vite, Tailwind 4 — `frontend/`
- **DB**: PostgreSQL, migrations in `migrations/`

Cursor rules in `.cursor/rules/` expand on structure; read those when editing matching files.

## Token-efficient workflow

### Before reading code

1. Identify the **domain** (inventory, pricing, identity, service_catalog, cost_report).
2. Use **grep** (`rg FuncName`, `rg "/api/v1/foo"`) to locate files—avoid repo-wide exploration.
3. Check **route list** in `internal/api/handler.go` `Router()` and **examples** in `rashnu.http`.
4. Do **not** read: `node_modules/`, `frontend/dist/`, `.idea/`, `go.sum`, generated artifacts.

### Minimal read set per task

| Task | Read |
|------|------|
| API behavior | `internal/api/<domain>.go`, `handler.go` routes |
| Business logic | `internal/<domain>/service.go`, `errors.go` |
| Data model | `internal/<domain>/types.go`, `dto.go` |
| SQL | `internal/<domain>/postgres_repository.go`, relevant `migrations/*.up.sql` |
| Frontend API | `frontend/src/lib/api.ts`, `mappers.ts`, `types.ts` |
| UI state | `frontend/src/state/useAppState.ts`, target `pages/*.tsx` |
| Wiring | `cmd/server/main.go` |

Read **types + dto first**—often enough to implement without loading entire services.

### Writing code

- **Smallest correct diff**; no unrelated refactors or new abstractions.
- **Follow existing file layout** in the domain package (see `.cursor/rules/go-backend.mdc`).
- **Mirror patterns**: sentinel errors, zap logging, chi handlers, pgx scan helpers.
- **Frontend**: API in `api.ts` only; mappers for shape conversion; state in `useAppState.ts`.
- **Tests**: only when asked or when covering real behavior; `make test-unit` for quick check.

## Architecture

```
cmd/server/main.go          → wires config, db, repos, services, HTTP server
internal/api/               → HTTP layer (Handler + per-domain handler files)
internal/<domain>/          → types, dto, errors, service, postgres_repository
internal/db/                → Postgres pool
internal/config/            → config.yaml loading
migrations/                 → SQL up/down pairs
frontend/src/lib/           → api, types, mappers, data, tweaks
frontend/src/state/         → useAppState hook
frontend/src/pages/         → UI pages
```

Auth: JWT Bearer; public `POST /api/v1/user/login`; admin role for writes (`RequireRole("admin")`).

## Commands

```bash
make dev              # Vite + go run backend
make migrate-up       # apply migrations
make build            # bin/server
make frontend-build   # frontend/dist (served by backend)
make test-unit        # go test -short
```

Config: `config.yaml`. Dev DB: see `Makefile` `DATABASE_URL` / `docker-compose.yml`.

## Adding an end-to-end feature

1. Migration (if needed) → domain types/dto/errors → repo → service → api handler + route → `main.go` wire-up.
2. Frontend: `api.ts` → `mappers.ts` → `useAppState.ts` → page UI.
3. Add entry to `rashnu.http` for manual API testing.

## Out of scope unless asked

- Commits, PRs, dependency upgrades, `.idea/` / IDE config changes.
- Reading or documenting `cost_calculation_analysis.md` unless working on cost logic.
