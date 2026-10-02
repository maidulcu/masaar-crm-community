# Security Policy

## Supported versions

Security fixes are released for the **latest tagged release** only. Please upgrade before reporting an issue you found on an older version.

## Reporting a vulnerability

**Please do not open a public issue for security problems.**

Report privately using either:

1. GitHub's **[Report a vulnerability](../../security/advisories/new)** button (Security tab → Advisories), or
2. Email **info@dynamicweblab.com** with the subject `SECURITY: masaar-crm-community`.

Please include the affected version/commit, steps to reproduce, and the impact you observed.

What to expect:

- Acknowledgement within **3 business days**.
- An initial assessment and severity within **7 days**.
- A fix and coordinated disclosure; we credit reporters who want to be credited.

## Scope

In scope: the Go API (`internal/`), the Next.js frontend (`web/`), the Docker/compose files and the migrations in this repository.

Out of scope: vulnerabilities in third-party services you connect (WhatsApp/Meta, SMTP providers, Ollama, etc.), social engineering, and denial-of-service through sheer traffic volume.

## Hardening checklist for self-hosters

- Generate a strong `JWT_SECRET` (`openssl rand -hex 32`). With `APP_ENV=production` the server refuses placeholder or short secrets.
- Set `ALLOWED_ORIGINS` to your frontend origin(s) — never `*`.
- If you enable WhatsApp, set both `WA_APP_SECRET` (so inbound webhooks are signature-verified) and a unique `WA_VERIFY_TOKEN`.
- Keep `ALLOW_REGISTRATION=false` (the default). Community Edition is designed for **one company per deployment**; do not expose open self-service signup.
- Terminate TLS in front of the API and the web app, and do not expose Postgres or Redis publicly.
- Run the provided containers as-is (they run as a non-root user) and keep images and dependencies up to date.
