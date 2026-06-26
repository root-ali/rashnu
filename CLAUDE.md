# Rashnu — Claude Code instructions

This file is the entry point for Claude Code. Full agent workflow: **AGENTS.md**. File-specific rules: **`.cursor/rules/*.mdc`**.

## Quick context

Go + React FinOps app for bare-metal cost tracking. Module `github.com/root-ali/rashnu`. Domains: `inventory`, `pricing`, `identity`, `service_catalog`, `cost_report`.

## Token rules (read this first)

1. **Narrow scope** — touch one domain; read only its `internal/<domain>/` files + matching `internal/api/<domain>.go`.
2. **Grep, don't wander** — `rg SymbolName` before opening files. Routes: `internal/api/handler.go`. API samples: `rashnu.http`.
3. **Skip** — `node_modules/`, `dist/`, `.idea/`, `go.sum`.
4. **Small diffs** — extend existing patterns; no drive-by refactors.

## Domain package layout

```
types.go  dto.go  errors.go  service.go  postgres_repository.go
```

API handlers live in `internal/api/`, not inside domain packages.

## Frontend layout

`api.ts` (HTTP) → `mappers.ts` (shape conversion) → `useAppState.ts` (state) → `pages/*.tsx` (UI). No raw `fetch` outside `api.ts`.

## Common commands

`make dev` · `make migrate-up` · `make build` · `make test-unit`

## Feature checklist

Migration → types/dto/errors → repo → service → handler + route → `main.go` → frontend api/mappers/state/page → `rashnu.http` entry.

Do not commit unless explicitly asked.
