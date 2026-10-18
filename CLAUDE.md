# CLAUDE.md

Masaar CRM, community edition: a multi-company real-estate CRM for the UAE. Go API (Fiber, pgx, PostgreSQL 16 + pgvector, Redis) and a Next.js web app. **This repo is public** — see "Public repo" below.

## Map

| Path | What |
| --- | --- |
| `cmd/server/main.go` | Wiring: config, repos, services, handlers |
| `internal/api/router.go` | Every route and its middleware (38 KB — grep it, don't read it whole) |
| `internal/api/handler/` | One file per area (`lead.go`, `public_lead.go`, `whatsapp*.go`, …) |
| `internal/api/middleware/` | JWT, API keys, roles, trial/suspension, quotas, demo guard |
| `internal/repo/` | All SQL. Only place that touches `pgxpool` |
| `internal/domain/models.go` | All domain structs (67 KB — grep for the type you need) |
| `internal/tenant/` | `tenant.From(ctx)` / `tenant.With(ctx, id)`: the company of the request |
| `internal/webhook/` | Outbound webhooks (`lead.*` events, HMAC-signed) |
| `migrations/` | goose SQL, `NNNN_name.sql`, next number = highest + 1, also above any open PR's migrations (`gh pr list`); goose refuses duplicate numbers. 0035 and 0041 are unused |
| `web/app/`, `web/lib/api.ts` | Next.js pages; API client (56 KB — grep it) |

Stubs in this edition (return 402/503, don't build on them): `internal/ai`, `internal/pdf`, `internal/billing/stripe.go`, `handler/ai.go`. Market data (`internal/bos24`, `handler/property.go`) is a real client for the waqov data API; off (503) without `BOS24_API_TOKEN`.

## Rules that matter

- **Multi-company.** Every company-owned table has `company_id NOT NULL`. Repositories call `tenant.From(ctx)` and filter by it; never trust a company id from the request body. A new repo method that skips the filter is a data leak.
- Handlers never run SQL; they call repos. Errors go through `handler/errors.go`.
- Write paths that act for a user call `AuditLogRepo.Log`; API-key paths call `LogAPIKey`.
- New behaviour is additive: a new migration, never an edit to an applied one. Down migrations must not delete user data.
- Public request fields are validated with explicit length/format limits (see `public_lead.go`).
- Update `CHANGELOG.md` (Unreleased) and, for API changes, `DEVELOPER.md` and `web/app/developers/page.tsx`.

## Commands

```bash
go build ./... && go vet ./... && go test ./...
TEST_DATABASE_URL=postgres://masaar:masaar@127.0.0.1:5432/masaar_test?sslmode=disable go test ./internal/repo/   # needs pgvector
cd web && npx tsc --noEmit && npm run build
```

The DB tests (`internal/repo/*_test.go`, incl. `tenant_isolation_test.go`) skip without `TEST_DATABASE_URL`; CI runs them on `pgvector/pgvector:pg16`.

## Keeping context small

- Don't read generated or bulky files: `docs/docs.go`, `docs/swagger.*`, `web/package-lock.json`, `go.sum`, `scripts/seed/main.go`. Grep them if you must.
- Grep before reading: `grep -n "func (r \*LeadRepo)" internal/repo/lead.go`, then read the line range you need.
- `CHANGELOG.md`, `API_INVENTORY.md`, `PROJECT_OVERVIEW.md`, `ROADMAP.md` are long; read only the section you need. `README.md` overstates features (it describes stubs as working).
- For a sweep across many files, use a search subagent and keep only its conclusion.

## Public repo

- No secrets, real hostnames, server IPs, customer names or internal plans in code, comments, commits or docs. `.gitignore` lists the private docs.
- Integrations stay generic: e.g. the lead intake takes `external_source` / `external_ref` and source `portal`, not a partner-specific field. Partner-specific glue lives in the partner's repo or in config.
