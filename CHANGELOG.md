# Changelog

All notable changes to Masaar CRM will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.3.4] - Unreleased

WhatsApp attachments (follow-up to v0.3.3). Includes migration `0062`.

### Fixed / added
- **Customer attachments are no longer lost.** Photos, documents and voice notes sent over WhatsApp are downloaded from Meta (which only keeps them ~30 days and hands out short-lived URLs), stored on the server and shown in the inbox. Files are fetched in the background when the message arrives and, if that failed, on demand when someone opens it. Expired files show a clear "no longer available" message.
- **Send files from your computer.** The inbox can now upload an image, PDF/Office document, audio or video to WhatsApp (previously only a public link could be sent). A copy is kept so the thread shows what was sent.
- **Captions are delivered.** The caption typed with an image/video/document was saved but never sent to the customer. Documents now carry their file name.
- Template messages without variables no longer fail (an empty body component was sent).
- The inbox shows errors from the send-media form instead of silently ignoring them.
- `docker-compose.prod.override.yml` defaulted `WA_BASE_URL` to `graph.instagram.com` (the Instagram API); it is now `graph.facebook.com`.

### Security
- **Customer-supplied files are treated as hostile.** They are served only through an authenticated, company-scoped endpoint; the declared type is never trusted (only images, audio, video and PDF are shown inline — everything else, including HTML/SVG, is a forced download), with `nosniff`, a sandboxing `Content-Security-Policy`, and a sanitised file name. Stored under random names (never a customer-chosen path) through `os.Root`, with a size cap (`WA_MEDIA_MAX_MB`, default 25).
- **The WhatsApp access token is only ever sent to Meta's own hosts.** The download URL comes from Meta's API response, so it is checked (https, Meta domains only, re-checked on redirects) and fetched through the SSRF-safe client.
- Outgoing uploads are limited to the file types WhatsApp accepts, size-capped and content-checked (a script renamed `.png` is refused); `media_type` is validated (it used to be written straight into the request as a JSON key).

### Upgrade notes
- Attachments are stored under `WA_MEDIA_DIR` (default `./data/whatsapp-media`; the Docker images use `/app/data` and the compose files mount a `media_data` volume there). Back this volume up with the rest of your data. If the directory cannot be created the integration keeps working and attachments are simply not shown.
- Messages received before this release have no stored file; their attachments are fetched from Meta on first open if still within Meta's 30-day window.

## [v0.3.3] - Unreleased

WhatsApp inbox fixes found in a pipeline audit. Includes migration `0061` (merges duplicate contacts — see upgrade notes).

### Fixed
- **Duplicate webhook deliveries no longer fail.** Meta redelivers a webhook until it gets a 2xx; a message that was already stored used to return HTTP 500, so Meta retried it for days and every later message in the same batch was lost. Processing is now idempotent, a bad item no longer stops the rest of its batch, and only genuine failures (e.g. database down) answer 5xx. A redelivery also no longer reopens a thread an agent has closed or inflates its message count.
- **Delivery receipts are applied.** Outbound messages now move through sent → delivered → read, and a failure (e.g. outside the 24-hour window, invalid number) is recorded with Meta's error code and shown on the message. Out-of-order receipts never move a status backwards.
- **One person, one contact.** Phone numbers are normalised to E.164 (`+971501234567`) everywhere they are written or looked up. WhatsApp reports numbers without the `+`, so a contact an agent created as `+971…` used to get a second contact and thread on the first inbound message. Contacts API/CSV import also accept `971 50 123 4567` and `00971…`.
- **Contact names are no longer overwritten** by the sender's WhatsApp profile name (or the bare number) on every message. A name is only filled in when it is empty or just the phone number.
- **Inbound media is no longer dropped.** Meta sends an image/document/voice-note *id*, not a URL, so these messages used to be skipped entirely. They now appear in the thread (`[Image] caption`, `[Document: passport.pdf]`, `[Voice message]`, …) with the media id and type stored for download. Locations, button/list replies, quick-reply buttons and contact cards keep their content instead of showing `[location]`; reactions are no longer stored as messages.
- **Agent replies are part of the conversation.** Sent messages (text, template, media) are stored in the thread, update its activity time and message count, show their delivery status, and are added to the lead timeline; inbound messages are too (when the contact has a lead). Previously AI summaries/drafts and inbox ordering only saw the customer's side.
- Auto-tagging of inbound messages ran without a company context and could never write; it now carries the company.
- `GET /threads/:id/messages?limit=` is bounded; messages over WhatsApp's 4096-character limit are rejected with 400.
- Message bodies are cleaned of NUL bytes/invalid UTF-8 so one malformed message cannot fail on every redelivery.

### Upgrade notes
- Migration `0061` normalises `contacts.phone_wa` to `+<digits>` and **merges contacts that differ only by formatting** within a company (oldest survives; leads, offers, viewings, timeline entries and WhatsApp threads move to it, nothing is deleted except the duplicate row). Numbers whose country cannot be determined (national format such as `0501234567`) are left untouched. The merge cannot be undone; back up before upgrading if you have many hand-entered duplicates.
- Importing a CSV or capturing a public lead for an existing number no longer renames the existing contact.

## [v0.3.2] - Unreleased

Third hardening pass: session storage and horizontal scaling. No schema changes.

### Security
- **Tokens out of `localStorage`** — the web app no longer keeps credentials where injected script can read them. The access token is held in memory only and the refresh token is an `HttpOnly` cookie (`masaar_rt`, path `/api/v1/auth`, `Secure` in production, `SameSite=Lax`). A reload restores the session from the cookie. Cookie mode is opt-in per request (`X-Auth-Mode: cookie`), so API, mobile and script clients keep using the refresh token in the JSON body exactly as before. Cookie-authenticated endpoints require the custom header (forces a CORS preflight) and an `Origin` in `ALLOWED_ORIGINS`; CORS now allows credentials for the configured origins. Tokens written to `localStorage` by earlier versions are deleted on first load.
- **Shared rate limiting** — IP rate limits (login, OTP, registration, refresh, webhooks, API) were counted per process, so running several API replicas multiplied every limit. Counters now live in Redis (fail-open if Redis is unreachable; authentication itself still depends on Redis and fails closed).
- **Content-Security-Policy** — now a full policy: `default-src 'self'`, scripts only from this site (plus Cloudflare Turnstile), `connect-src` restricted to this site, the API and the WebSocket, and `frame-ancestors/base-uri/object-src/form-action` locks. `script-src` still allows `'unsafe-inline'` (Next.js bootstrap scripts); a nonce-based policy remains future work.

### Fixed
- **Import / export downloads** — the Import/Export page read the token from the wrong storage key and put it in the URL (`?token=`, `?authorization=`), which the API ignores — downloads failed with 401 and the credential leaked into history and logs. Downloads and the listing brochure/QR links now use authenticated fetches. Wrong-password responses on the login page no longer trigger a pointless refresh attempt and page reload.
- `gofmt` is now enforced in CI.

### Upgrade notes
- **The web app and the API must be on the same site** (same registrable domain, e.g. `crm.example.com` and `api.example.com`, or `localhost:3000` and `localhost:8080`) for the refresh cookie to be sent. For different sites set `AUTH_COOKIE_SAMESITE=none` (HTTPS required).
- `ALLOWED_ORIGINS` now defaults to `http://localhost:3000` instead of `*` outside production; credentialed CORS cannot be used with `*`, so the browser app needs explicit origins.
- Everyone is signed out once on upgrade (tokens move from `localStorage` to the cookie).
- Running more than one API replica? They now share limits through the same Redis automatically.

### Known limitations
- WhatsApp, SMTP and AI credentials are still configured per deployment, not per company.
- `script-src` still permits inline scripts (see above).

---

## [v0.3.1] - Unreleased

Second security audit pass over the merged v0.3.0 code. No schema changes.

### Security
- **Sessions** — deactivating or deleting a user, changing their role, or changing/resetting their password now ends all of their existing logins immediately (per-user revocation cutoff in Redis, enforced by the auth middleware and by refresh). Refresh previously did not check that the user was still active, so a deactivated account could renew its session indefinitely; it now also checks the company is active.
- **Stored XSS** — URL fields (`*_url`, `url`) in API request bodies must be http(s) URLs or same-site paths; `javascript:`, `data:` and similar are rejected with 422. The web app additionally sanitises every stored URL it renders as a link, image or video (`safeUrl`) and sets `rel="noopener noreferrer"` on external links.
- **Rate-limit bypass** — client IPs were taken from the first (client-controlled) entry of `X-Forwarded-For`, so every IP-based limiter (login, OTP, registration) could be evaded by spoofing it. The API now reads a single-valued header the proxy overwrites (`PROXY_HEADER`, default `X-Real-IP`) and the provided nginx configs overwrite `X-Forwarded-For`.
- **SSRF** — the DocuSign document fetch used a plain `http.Get` on a user-supplied URL (and read the response without a limit). It now goes through a shared SSRF-safe client (`internal/safehttp`) that refuses internal addresses — including carrier-grade NAT, `0.0.0.0/8`, multicast, NAT64 and IPv4-mapped IPv6 — checks every redirect hop, allows only http(s) and caps the size. Webhook delivery uses the same client.
- **Information disclosure** — about 200 handlers returned raw `err.Error()` text (SQL and driver messages) to clients. Errors are now logged server-side and answered with generic messages and accurate status codes (404 not found / foreign reference, 409 duplicate, 422 invalid data).
- **Authorization** — contact/lead exports require an admin or agent role; only admins can browse other agents' commissions; a contact's `assigned_to` must belong to the caller's company; list endpoints bound `page`/`limit`.
- **Login** — email is normalised so lockout counters cannot be bypassed by changing case; unknown emails cost the same bcrypt time as real ones (no timing enumeration); the magic-link endpoint only emails registered, active accounts (it could be used to send unsolicited mail to arbitrary addresses); password-reset tokens are never printed in production logs.
- **Exports** — CSV exports neutralise spreadsheet formula injection.
- **Deployment** — Docker Compose published Postgres, Redis and Ollama on all interfaces (Redis without authentication); they are now bound to `127.0.0.1`, as are the API and web ports in the production override. In production the server also refuses a placeholder or well-known database password. The Next.js app now sends `Content-Security-Policy` (`frame-ancestors`, `base-uri`, `object-src`, `form-action`), `X-Frame-Options`, `X-Content-Type-Options`, `Referrer-Policy`, `Permissions-Policy` and `Cross-Origin-Opener-Policy`, and no longer sends `X-Powered-By`.

### Upgrade notes
- **Reverse proxy:** make sure your proxy *overwrites* the client-IP header. The updated `nginx.conf.template` and `docker/nginx.conf` do (`X-Real-IP` and `X-Forwarded-For` set to `$remote_addr`). Behind Cloudflare set `PROXY_HEADER=CF-Connecting-IP`. Without a proxy overwriting the header, the app falls back to the connecting address.
- Users whose role or password changes, or who are deactivated, must sign in again (by design).
- Compose users who connected to Postgres/Redis from another machine must now tunnel (e.g. SSH) or change the bindings deliberately.
- Existing rows containing non-http(s) values in URL fields are not rewritten; the UI will render those links inert.

### Known limitations
- Access and refresh tokens still live in browser `localStorage` (fixed in v0.3.2).

---

## [v0.3.0] - Unreleased

First release prepared for public use. Contains security hardening, build fixes and **behaviour changes** (see "Upgrade notes").

### Security
- **Company isolation** — contacts, leads, deals, invoices, WhatsApp threads/messages, notifications, email history, viewings, offers, communication history, audit logs and integration settings are now scoped to the owning company (migration `0060`, `company_id` backfilled from each row's owner/parent). Every repository query filters by the caller's company, creates ignore any client-supplied company id and verify that referenced records (contact, lead, listing, property, tenant, …) belong to the same company, and company settings no longer return "the first row". The company travels in the request context (`internal/tenant`) and fails closed when absent. Covered by integration tests that run against Postgres in CI.
- **WebSocket** — connections are now bound to the identity in the verified access token (the client-supplied `user` query parameter is ignored) and events are delivered only within the sender's company. The browser connects with `?token=<access token>`; revoked tokens are rejected on this path too.
- **JWT** — access tokens are now required to carry the `masaar-crm` audience (the claim was previously issued but not checked) and only HS256 is accepted.
- **Users API** — updating, deactivating and deleting a user is now restricted to the caller's own company.
- **SMS OTP** — verification is capped at 5 attempts per phone number per 15 minutes, compared in constant time and consumed atomically. Magic-link tokens are consumed atomically.
- **Rate limiting** — added limiters to OTP/magic-link verification, password reset and token refresh.
- **WhatsApp webhook** — in production, inbound webhooks are rejected unless `WA_APP_SECRET` is configured so signatures can be verified.
- **Startup validation** — `config.Validate()` refuses placeholder/short `JWT_SECRET`, wildcard `ALLOWED_ORIGINS` and an unsigned/default-token WhatsApp setup when `APP_ENV=production`.
- **Containers** — the root `Dockerfile` and `web/Dockerfile` now run as a non-root user; base images pinned (`alpine:3.22`).
- **Go dependencies** — `govulncheck` (now run in CI) flagged 42 reachable vulnerabilities. Go toolchain raised to 1.25.13 (standard library fixes) and Fiber 2.52.4 → 2.52.12, golang-jwt 5.2.1 → 5.2.2, go-redis 9.5.1 → 9.6.3, pgx 5.8.0 → 5.9.2, plus `x/net`, `x/text` and `x/crypto`.
- **Dependencies** — Next.js 15.5.15 → 15.5.27 (clears the critical advisories) and transitive `nanoid`/`sharp` fixes. Removed the unused `react-leaflet` dependency (it required React 19 and broke `npm install`).
- Added `SECURITY.md`, Dependabot and CI (vet, build, test, `govulncheck`, `npm audit`, secret scan).

### Fixed
- **Server entry point** — `cmd/server/main.go` is restored (it was never committed because of the `.gitignore` pattern below), so the API can be built and the Docker image works.
- **Fresh installs** — migration `0059` referenced a non-existent `audit_logs.created_at` and aborted every new database; the compose files used `postgres:16-alpine`, which lacks the `pgvector` extension that migration `0002` needs.
- **Inbound WhatsApp & new companies** — contact upsert, thread upsert and company-settings reads failed on NULL columns; API-key lead intake panicked (company id was stored as a UUID but read as a string); `ListingRepo.GetByID` failed on every call; the commission lease query referenced columns that do not exist; dashboard stats counted soft-deleted leads.
- **Swagger UI** — in production it was protected by the first 16 characters of `JWT_SECRET`; it now uses a dedicated `DOCS_PASSWORD` and is disabled when unset.
- **Demo seed** — `scripts/seed` ran with several errors against the current schema; it now runs cleanly and idempotently.
- **Repository** — `cmd/server/` was silently excluded by a `.gitignore` pattern; the pattern is now anchored so the entry point can be tracked. A 13 MB compiled `seed` binary and `tsconfig.tsbuildinfo` were removed from version control.
- **Frontend build** — Next.js 15 `params` typing on the public listing page and the missing `back` prop on `Header`; `next build` now succeeds.
- **Docker** — healthchecks used `curl`, which the images do not ship; the web healthcheck pointed at a missing `/api/health` route (added); `NEXT_PUBLIC_*` values are now passed as build args (they are inlined at build time, so runtime env had no effect); `BOS24_TOKEN` → `BOS24_API_TOKEN`; default `WA_BASE_URL` pointed at the wrong host; `WA_APP_SECRET`, `ALLOWED_ORIGINS`, `APP_URL` and `ALLOW_REGISTRATION` were never passed through to the container.
- **Real-time notifications** — the browser WebSocket could not authenticate (the API required an `Authorization` header that browsers cannot send on WebSocket connections).
- **Docs** — corrected the clone path and removed a default login (`admin@masaar.local` / `changeme`) that no migration or seed created.

### Upgrade notes
- `ALLOW_REGISTRATION` now defaults to **`false`**. On a fresh install signup stays open until the first user is created (that account adopts the company in `APP_COMPANY_ID`), then closes. Set it to `true` to host several companies.
- Migration `0060` attributes existing rows to a company through their owner/parent, else to the only company that has users, else the seed company. If you previously ran with open signup and several companies, review the result — shared rows cannot be split retroactively.
- Behind nginx, set `TRUSTED_PROXIES` so per-client rate limits see real client IPs.
- Existing sessions must sign in again once (older access tokens lack the audience claim).
- With `APP_ENV=production` the server will not start with placeholder secrets — set `JWT_SECRET` and `ALLOWED_ORIGINS`, plus `WA_APP_SECRET` / a unique `WA_VERIFY_TOKEN` if WhatsApp is enabled.
- `NEXT_PUBLIC_*` variables are build-time: rebuild the web image after changing them.

### Known limitations
- Deployment-wide settings are shared by all companies: the WhatsApp Cloud API number, SMTP and AI configuration come from environment variables, and inbound WhatsApp messages are routed to `APP_COMPANY_ID`. Per-company messaging credentials are future work.
- Build-time PostCSS advisories remain until Next.js 16 (a major upgrade).

---

## [v0.2.1] - 2026-09-11

### Security Fixes
- **API Key Rate Limiter** — Fixed non-atomic INCR + EXPIRE race condition with Lua script
- **Password Reset** — Fixed TOCTOU vulnerability with atomic UPDATE ... RETURNING
- **Email Validation** — Added format validation to forgot-password endpoint
- **JWT Claims** — Added `aud` (audience) claim for token type validation
- **Webhook Retry** — Increased context timeout from 30s to 60s for 3-retry attempts

### Performance
- **Added 4 production indexes** for common query patterns:
  - Lease date-range queries (renewal, overdue)
  - Payment overdue detection
  - Audit log time-range queries
  - Lead soft-delete filtering

### Bug Fixes
- Fixed potential race conditions in distributed rate limiting
- Improved token validation security posture
- Enhanced webhook reliability under network delays

---

## [v0.1.0] - 2026-03-13

### Added
- **WhatsApp Receiver** — Receive and read incoming WhatsApp messages via Meta webhook
- **Sales Pipeline** — Single Kanban board with drag-drop stage management (dnd-kit)
- **AI Summarize** — Manual "Summarize" button powered by Ollama (local LLM, data never leaves server)
- **Deals Management** — Deal list, stage tracking, linked to leads
- **VAT Invoicing** — Invoice generator with 5% UAE VAT; sequential `INV-YYYY-NNNN` numbering; draft → sent → paid workflow
- **Notifications** — Personal real-time notifications via WebSocket (`/ws/notifications`)
- **Keyword Search** — Search contacts by name, phone, email
- **Contact Management** — Unified contact profiles linked to WhatsApp threads
- **Audit Log** — Immutable activity log for PDPL compliance
- **RTL / LTR** — Full Arabic interface; Cairo font for AR, Inter for EN; logical CSS properties
- **RBAC** — Role-based access control (admin / agent / viewer) enforced on all write routes
- **Rate Limiting** — 300 req/min on WhatsApp webhook; 10 req/min on login
- **OpenAPI / Swagger UI** — Auto-generated spec at `/docs`, `docs/swagger.yaml` committed to repo
- **Self-hosted** — Full stack via `docker compose up` (Next.js + Go + PostgreSQL + Redis + Ollama)

### Tech Stack
- Go 1.22 + Fiber v2
- Next.js 14 + Tailwind CSS (App Router, TypeScript)
- PostgreSQL 16 + pgvector
- Ollama (llama3 / mistral)
- WebSockets (native Fiber)
- Redis (refresh tokens + blacklist)
- goose v3 (SQL migrations)

---

## [v0.2.0] - 2026-04-26

### Phase 15: Core Operations & Analytics

#### Phase 15.1: Expense Tracking System
- **Added** comprehensive expense management system
  - `expense_categories` table with type enum (maintenance, utilities, insurance, repairs, staff, cleaning, other)
  - `expenses` table with full expense lifecycle (amount, date, vendor, payment status, receipt tracking)
  - `expense_approvals` table for approval workflow
  - 8 API endpoints: category management, expense CRUD, approval workflow
  - Role-based access: Admin full access, Agent create/update
  - Real-time notifications on approval/rejection

#### Phase 15.2: Inspection & Maintenance Scheduling
- **Added** inspection and maintenance management system
  - `inspection_templates` table with reusable checklists
  - `inspections` table with status tracking, findings, severity levels, photos
  - `maintenance_tasks` table with priority, contractor tracking, cost estimation
  - `maintenance_photos` table for before/during/after documentation
  - 16 API endpoints for inspection and maintenance workflows
  - Frontend pages: `/inspections`, `/maintenance` with list, filter, detail views
  - Features: Priority-based sorting, status tracking, photo gallery

#### Phase 15.3: Tenant Analytics Dashboard
- **Added** comprehensive analytics system
  - Company-wide KPIs: occupancy rate, revenue, collection rate
  - Property-level metrics: revenue, NOI, maintenance needs
  - Tenant performance metrics: risk scoring based on payment history, disputes, tenure
  - Financial analytics: revenue trends, expense breakdown, profit margins
  - Maintenance metrics: task completion rates, response times
  - 7 API endpoints under `/api/v1/analytics/`
  - Frontend analytics pages:
    - `/analytics/overview` - KPI dashboard
    - `/analytics/properties` - Property performance
    - `/analytics/tenants` - Risk scoring
    - `/analytics/financial` - Revenue analysis
  - Features: Real-time calculations, bilingual labels, color-coded risk scores

### Phase 16: Advanced Workflows

#### Phase 16.1: Lease Renewal Automation
- **Added** automated lease renewal workflow system
  - `lease_renewal_workflows` table with status tracking, proposed terms, counter-offers
  - `renewal_communication_templates` table for email/WhatsApp templates (bilingual)
  - `renewal_communication_log` table for audit trail
  - 9 API endpoints: initiate, propose, send-offer, accept, reject, counter-offer, templates
  - Features: Automated renewal tracking, communication history, tenant responses
  - Role-based access: Admin initiates, Agent/Admin manage workflow

#### Phase 16.2: Commission Tracking & Agent Performance
- **Added** database schema (Migration 00027):
  - `commission_structures` table (fixed/percentage/tiered types)
  - `agent_commissions` table (monthly tracking)
  - `commission_transactions` table (individual entries)
  - Domain models and enums ready for handler implementation

#### Phase 16.3: Document Management & e-Signature
- **Added** database schema (Migration 00028):
  - `document_templates` table (lease, offer, inspection, waiver, custom)
  - `documents` table with signature status and audit trail
  - `document_signatures` table with IP/user-agent logging
  - `document_audit_log` table for compliance
  - Data classification: public, internal, confidential
  - Ready for e-signature integration (DocuSign framework)

#### Phase 16.4-16.6: Infrastructure
- **Added** custom fields system (Migration 00029):
  - Entity-specific custom metadata support
  - Dynamic field storage and validation
- **Added** bulk operations framework (Migration 00030):
  - CSV import/export job tracking
  - Error logging and data validation
  - Dry-run mode support

### Infrastructure Updates
- Total migrations: 30 (up from previous)
- New tables: 35+ across all phases
- Total API endpoints: 100+ with consistent response format
- All endpoints include role-based access control
- Full audit trails for sensitive operations

### Breaking Changes
- None. All changes are backward-compatible.

### Known Limitations
- DocuSign e-signature not yet implemented (framework ready)
- Commission calculation engine pending implementation
- Bulk import/export job processing pending
- Analytics caching in Redis not yet implemented

---

## [v0.3.0] — 2026-05-27

### Critical Bug Fixes

#### Backend: `uuid.Parse` compile error across 5 handler files
- **Root cause:** `uuid.Parse(str)` returns `(uuid.UUID, error)` but all 14 call-sites across 5 files captured only one return value, preventing the entire binary from compiling.
- **Files fixed:** `document.go`, `expense.go`, `inspection.go`, `lease_renewal.go`, `maintenance.go`
- **Fix:** All occurrences changed to `id, _ := uuid.Parse(...)` or `field, _ = uuid.Parse(...)`.
- **Impact:** Documents, Expenses, Inspections, Lease Renewals, and Maintenance were completely non-functional.
- **See:** [Development Notes — uuid.Parse Pattern](#development-notes) in CLAUDE.md

### Added

#### Message Templates Management Page (`/message-templates`)
- Full CRUD frontend page for WhatsApp message templates (backend was already complete)
- Table view: Name, Category, Variables, Active status
- Create/Edit modal with `{{1}}` `{{2}}` variable placeholder syntax, active toggle
- Click-to-preview panel renders a WhatsApp-style bubble with variables substituted
- Sidebar: added under Inbox in CRM section

#### Email History Page (`/email-history`)
- Standalone page listing all sent emails across all entities
- Status filter chips: All / Sent / Failed / Bounced / Pending
- Click-to-detail panel: full body, error message if failed, timestamps
- **New backend:** `EmailRepository.ListAll(ctx, page, limit)` + `GET /api/v1/emails`
- Sidebar: added in CRM section

#### Invoices List & Detail Pages (`/invoices`, `/invoices/[id]`)
- List page: summary cards (Subtotal / VAT (5%) / Total), status filter, inline PDF link
- Detail page: line-item breakdown, status action buttons (Mark Sent / Mark Paid), link to deal
- **New backend:** `InvoiceRepo.ListAll(ctx, page, limit)` + `GET /api/v1/invoices`
- Sidebar: added in CRM section

#### AI Lead Scoring on Contact Detail (`/contacts/[id]`)
- "🤖 AI" button beside the lead score badge; visible to Agent/Admin when contact has leads
- Calls Ollama LLM, parses `{"score": int, "reasoning": "..."}` response
- Persists score back to `contacts.lead_score` immediately; updates display live
- Shows AI reasoning text inline below the badge
- **New backend:** `ContactRepo.UpdateScore(ctx, id, score)` + `POST /api/v1/ai/score-contact/:id`
- Note: Pipeline page already had AI scoring — this completes the contact detail page

#### Renewal Templates Management (`/renewal-templates`)
- Card-grid CRUD page (Admin only) for renewal communication templates
- Card shows: template name, language badge, email subject preview, WhatsApp message excerpt
- Create/Edit modal with Email / WhatsApp tab switcher; supports bilingual templates
- **New backend:** `RenewalTemplateRepo.Update` + `RenewalTemplateRepo.Delete` + `PATCH /renewal-templates/:id` + `DELETE /renewal-templates/:id`
- Sidebar: added in Properties section (admin only)

### New API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/v1/invoices` | List all invoices, paginated |
| `GET` | `/api/v1/emails` | List all email history, paginated |
| `POST` | `/api/v1/ai/score-contact/:id` | AI-score a contact via their latest lead; persists score |
| `PATCH` | `/api/v1/renewal-templates/:id` | Update renewal template |
| `DELETE` | `/api/v1/renewal-templates/:id` | Delete renewal template |

### New Frontend Routes

| Route | Notes |
|-------|-------|
| `/message-templates` | Full CRUD, Agent + Admin |
| `/email-history` | Read-only log, Agent + Admin |
| `/invoices` | List with summary cards |
| `/invoices/[id]` | Detail + status actions + PDF download |
| `/renewal-templates` | Full CRUD, Admin only |

### Known Gaps Identified (2026-05-27 audit)

All gaps below were found during a systematic audit and either fixed or documented:

| # | Gap | Severity | Resolution |
|---|-----|----------|------------|
| 1 | `uuid.Parse` single-value assignment in 5 handler files — backend won't compile | **Critical** | Fixed in this release |
| 2 | `GET /api/v1/invoices` missing — no standalone invoices list page possible | High | Fixed in this release |
| 3 | `GET /api/v1/emails` missing — email history page not possible | High | Fixed in this release |
| 4 | AI scoring only covered leads, not contact detail page | High | Fixed in this release |
| 5 | `RenewalTemplateRepo` missing `Update` + `Delete` | Medium | Fixed in this release |
| 6 | Message Templates page entirely missing (backend was done) | Medium | Fixed in this release |
| 7 | `ai.go` missing `"log"` import | Medium | Fixed in this release |

---

## [Unreleased]

### Phase 17: SaaS Multi-Tenant Platform (Coming Soon)
- Company onboarding workflow
- Subscription management
- Usage analytics and billing
- Tenant isolation at API and database level

### Phase 18: Advanced Integrations (Coming Soon)
- DocuSign e-signature
- Stripe/PayPal payment processing
- Accounting software integrations
- SMS gateway expansion

### Phase 19: AI & Automation (Coming Soon)
- Automated lease renewal workflows
- Predictive tenant risk scoring
- Expense categorization via ML
- Document OCR and extraction

---

*For self-hosters: Always check the changelog before upgrading. Breaking changes will be marked with ⚠️.*
**Status**: Phases 15-16 complete, ready for Phase 17 planning.
**Maintainer**: Dori Internet Dev Team
