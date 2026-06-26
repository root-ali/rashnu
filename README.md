# Rashnu

FinOps tool for bare-metal infrastructure. You keep inventory (datacenters, servers, network gear), attach workloads from your service catalog, layer on depreciation pricing, and get per-service cost reports.

Go API + React frontend. PostgreSQL for storage.

## Requirements

- Go 1.26+
- Node.js (for the frontend)
- PostgreSQL 16

## Local development

Point `config.yaml` at your database. Example DSN:

```
postgres://rashnu:rashnu@localhost:5433/rashnu?sslmode=disable
```

Migrations run automatically when the server starts.

```bash
go run ./cmd/server          # API on :8080

cd frontend && npm install && npm run dev   # Vite on :5173
```

Auth is JWT — `POST /api/v1/user/login` to get a token; admin role needed for writes.

## Docker

Full stack (API, Postgres, Prometheus):

```bash
docker compose up --build
```

App on http://localhost:8080. Migrations run from `./migrations` on first Postgres start.

## License

MIT — see [LICENSE](LICENSE).
