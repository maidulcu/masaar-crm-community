# Contributing to Masaar CRM

Thank you for your interest in contributing to Masaar CRM — the open-source WhatsApp CRM built for the UAE.

## What's in scope

The community edition includes: WhatsApp pipeline, lead and contact management, deals, rentals, leases, invoices (CRUD), Arabic RTL interface, and self-hosted deployment.

**Pro features** (AI scoring, PDF generation, BOS24 market data) live in a separate private repo and are not accepting community contributions at this time.

## Getting started

### Prerequisites

- Go 1.22+
- Node.js 18+
- Docker + Docker Compose
- PostgreSQL 16 with pgvector

### Local setup

```bash
git clone https://github.com/maidulcu/masaar-crm-community.git
cd masaar-crm-community
cp .env.example .env        # fill in your values
docker compose up -d postgres redis
go run ./cmd/server          # backend on :8080
cd web && npm install && npm run dev   # frontend on :3000
```

Swagger docs: http://localhost:8080/docs  
First run: create your admin account at http://localhost:3000/signup. For demo data, run `go run ./scripts/seed` (see `scripts/seed/README.md` for the demo logins).

## Submitting a pull request

1. Fork the repo and create a branch from `main`
2. Keep changes focused — one feature or fix per PR
3. Run `go vet ./...` and `npm run lint` before opening the PR
4. Describe what you changed and why in the PR description
5. If you're fixing a bug, include steps to reproduce

## Code style

- **Go**: `gofmt` formatted, follow existing patterns in `internal/`
- **TypeScript**: ESLint via `npm run lint`, Tailwind for styling
- **Commits**: short present-tense summary (`fix: magic link token expiry`)
- **No AI-generated code** without human review and testing

## Reporting issues

Use [GitHub Issues](https://github.com/maidulcu/masaar-crm/issues) with:
- A clear title
- Steps to reproduce (for bugs)
- Expected vs actual behaviour
- Go version, OS, and Docker version if relevant

## Sensitive data

Never include real credentials, API keys, or production data in PRs or issues. Redact anything sensitive before sharing logs or config.

## License

By contributing you agree that your contributions will be licensed under the [MIT License](LICENSE).
