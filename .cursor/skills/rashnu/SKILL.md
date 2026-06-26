---
name: rashnu
description: >-
  Work in the rashnu FinOps codebase with minimal token use. Use when implementing
  features, fixing bugs, or exploring inventory, pricing, service catalog, cost
  reports, identity, or the React frontend.
---

# Rashnu project skill

## When to use

Any task in this repo: backend domains, API, migrations, or React frontend.

## Workflow

1. Read **AGENTS.md** (workflow) — not the whole codebase.
2. Pick domain from the map in `.cursor/rules/rashnu-core.mdc`.
3. `rg` for the symbol/route/table name.
4. Read the minimal file set from AGENTS.md table.
5. Implement matching existing patterns in that domain.
6. Run `make test-unit` or `go build ./...` when backend changes; `cd frontend && npm run build` for FE.

## Domain → files

- **inventory**: datacenters, servers, infrastructure hardware
- **pricing**: depreciation, daily/monthly unit prices
- **service_catalog**: services, pod/vm workloads, prometheus config
- **cost_report**: daily/monthly reports, trends, calculate endpoint
- **identity**: users, JWT login/logout

## Anti-patterns

- Broad codebase search or reading all handler files
- New repository interfaces when `PostgresRepository` concrete type is the norm
- `fetch` in React components
- Editing old migrations instead of adding new ones
- Large refactors across domains in one change

## Reference files

| Need | File |
|------|------|
| All routes | `internal/api/handler.go` |
| DI wiring | `cmd/server/main.go` |
| API examples | `rashnu.http` |
| HTTP client | `frontend/src/lib/api.ts` |
| App state | `frontend/src/state/useAppState.ts` |
