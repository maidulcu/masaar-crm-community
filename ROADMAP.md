# Masaar CRM — Product Roadmap

Last updated: 2026-05-04

---

## Status legend
- ✅ Done & merged
- 🔄 In progress (on `claude/review-uae-launch-Zvbqq`)
- 🗓 Planned — next
- 💡 Planned — later
- 🤝 Partnership idea

---

## Completed (this branch)

### Auth & Security ✅
- Email normalisation on create + lookup (lowercase, trimmed)
- Password strength validator (8+ chars, uppercase, digit) on all write paths
- Account lockout after 5 failed attempts (Redis, 15-min window)
- Single-use refresh tokens (`GetDel` — replay protection)
- Full token rotation on refresh (new access + refresh pair)
- Active-user check on refresh (deleted accounts get 401)
- Fail-closed logout (errors on revoke → 500, not silent pass)
- Access token blacklist on logout
- Forgot-password flow: SHA-256 hashed reset token, 1-hour TTL, SMTP or dev stdout fallback
- Password reset: single-use token, strength check, audit log
- Login audit log with client IP
- Startup validation: fatal if JWT_SECRET is default or ALLOWED_ORIGINS is `*` in production

### User Management ✅
- `GET /users` — admin-only user list
- `POST /users` — admin-only user creation with validation
- `PATCH /users/me/password` — self-service password change with audit log

### Lead Pipeline ✅
- Soft delete (`deleted_at`) — leads never hard-deleted
- `assigned_to` — leads assignable to specific users
- `closed_reason` — captured on Won/Lost stage changes
- `last_contacted_at` field
- `GET /leads/search` — filtered list with stage/source/assignee/query params
- `PATCH /leads/:id/assign` — reassign a lead
- `DELETE /leads/:id` — admin soft delete
- Webhook events fired on: `lead.created`, `lead.stage_changed`, `lead.won`, `lead.lost`

### API Keys ✅
- `sk_live_[48hex]` format, SHA-256 hashed in DB
- Scope-based access control (`lead:create`, etc.)
- Per-key Redis rate limiting (300 req/min sliding window)
- Admin CRUD: list, create (plaintext shown once), revoke

### Outbound Webhooks ✅
- HMAC-SHA256 signed payloads (`X-Masaar-Signature: sha256=...`)
- Async goroutine dispatch, 3 retries with 1s/4s backoff
- Delivery log stored in DB (success/failure, status code)
- Admin CRUD: list, create (secret shown once), delete, test

### Public Lead Intake ✅
- `POST /webhooks/leads` — API key auth, scope `lead:create`
- E.164 phone validation, contact upsert (deduplication)
- Property metadata enriched into notes
- Fires `lead.created` webhook

### BOS24 Real Estate Integration ✅
- **Critical fix**: POST body was `nil` — `SearchProperties` always sent empty requests
- 15 new client methods: Areas, POIs, Ejari rentals/yield, Developers, Projects, AI describe, Valuations, Units, Transaction heatmap
- 18 handler endpoints covering full BOS24 dashboard surface
- Response caching in Redis (5min–24h by data type)
- Graceful 503 when BOS24 token not configured

### Infrastructure ✅
- PII-safe access logger (uses `${path}` not `${url}` — no phone numbers logged)
- nginx config with TLS, HSTS, WebSocket upgrade, Swagger restricted to internal IPs
- Production CORS and JWT secret validation at startup
- Error handler: 500s log internally, return generic message externally
- Gemini AI provider alongside Ollama (`AI_PROVIDER=gemini`)

---

## Planned — Next Sprint 🗓

### 0. Plans, Billing & Quota System ✅ (just shipped)
Full subscription plan infrastructure with Stripe integration.

**Plans:**
| Plan | Price | BOS24/mo | AI/mo | PDFs/mo |
|------|-------|----------|-------|---------|
| Community | Free | 0 | 0 | 0 |
| Starter | $29 | 200 | 100 | 10 |
| Pro | $79 | 1,000 | 500 | Unlimited |
| Business | $199 | Unlimited | Unlimited | Unlimited |

**What's built:**
- Migration `00036`: `plan`, `stripe_customer_id`, `stripe_sub_id`, `plan_expires_at` on `company_settings` + `usage_tracking` table
- `internal/billing/plans.go` — plan definitions, quota check helpers
- `internal/billing/stripe.go` — Stripe Checkout, Customer Portal, webhook parsing
- `internal/repo/billing.go` — atomic usage increment (`INSERT ... ON CONFLICT DO UPDATE`), usage reads
- `internal/api/middleware/quota.go` — `CheckQuota(resource)` + `CheckUserAIQuota()` (per-user daily AI cap)
- `internal/api/handler/billing.go` — `GET /billing`, `POST /billing/checkout`, `POST /billing/portal`, `POST /webhooks/stripe`
- Quota middleware wired to all BOS24 and AI routes
- Billing page at `/settings/billing` — current plan card, usage meters with progress bars, plan comparison grid, Stripe Checkout redirect
- Sidebar "Plans & Billing" link

**Env vars needed for Stripe:**
```bash
STRIPE_SECRET_KEY=sk_live_...
STRIPE_WEBHOOK_SECRET=whsec_...
STRIPE_PRICE_STARTER=price_...
STRIPE_PRICE_PRO=price_...
STRIPE_PRICE_BUSINESS=price_...
```
*(Leave empty for self-hosted/community — works without Stripe)*

---

### 1. BOS24 Dashboard Widgets (Frontend)
Wire the new `/api/v1/properties/*` endpoints into interactive dashboard components.

| Widget | Endpoint | Notes |
|--------|----------|-------|
| Area price heatmap | `GET /properties/transactions/areas` | Choropleth map by avg AED/sqft |
| Yield calculator | `GET /properties/rentals/ejari/yield` | Input: area + type → shows % ROI |
| POI proximity card | `GET /properties/pois` | Metro, malls, schools near a lead's area |
| Market trend chart | `GET /properties/market-trends` | 12-month sales vs rental trend line |
| Area comparison | `GET /properties/areas/:slug/summary` | Side-by-side area stats for pitch prep |
| AI listing writer | `POST /properties/ai/describe` | Generate marketing copy from unit details |

All visible on the lead detail page under a "Market Research" tab — agent can open with one click before a client call.

### 2. Property Research Report — PDF
Generate a branded PDF to send clients after a research session.

**Structure:**
1. Company header (logo, agent name, date)
2. Client brief (name, requirements: area, budget, type, bedrooms)
3. Area market overview (avg price/sqft, YoY change, transaction volume)
4. Comparable transactions table (last 10 DLD deals in area)
5. Rental yield summary (if investor profile)
6. Nearby amenities (top 5 POIs — metro, schools, malls)
7. AI-generated property description (if specific unit)
8. Legal disclaimer + company branding footer

**Implementation path:**
- `POST /api/v1/properties/report/pdf?lead_id=X` — pulls lead's area preference, assembles PDF
- Extend `internal/pdf/` (already used for invoices) with a `PropertyReport` type
- "Generate Client Report" button on lead detail page → download / email to client

### 3. Minor Auth Gaps
- Email format validation (regex) on `POST /users` and `POST /auth/forgot-password`
- `aud` claim in JWT access tokens (`"aud": "masaar-crm"`)
- Fix TOCTOU race in `ConsumePasswordResetToken` — mark used and read in single atomic UPDATE ... RETURNING

### 4. Fix API Key Rate Limiter Race
- Current `Incr` + conditional `Expire` is non-atomic
- Replace with Lua script or `SET NX EX` + `INCR` pattern to prevent keys that never expire

---

## Planned — Later 💡

### 5. Push Listings to BOS24 (Agent OS Feature)
Allow agents to publish a property listing from Masaar directly to BuyOrSell24 with one click.

**Flow:**
1. Agent fills in listing form (unit, area, price, photos, description)
2. AI auto-generates listing description via `POST /properties/ai/describe`
3. "Publish to BOS24" button → `POST /api/v1/properties/publish` → BOS24 listing API
4. Status shown on listing card (published / pending / rejected)
5. Webhook from BOS24 on view/enquiry events → notification in Masaar

**Value:** Masaar becomes the single workspace for UAE agents — manage CRM + publish listings without switching tools.

### 6. AI Listing Health Score (Gemini)
Before publishing a listing, score it on predicted time-to-sell:

- Photo quality (min 5 photos, outdoor + indoor)
- Price vs area average (under/overpriced %)
- Description completeness (Arabic + English, key amenities mentioned)
- POI coverage (mentions metro/school proximity)

Score 0–100 shown with actionable tips. Uses Gemini Vision for photo analysis.

**API:** `POST /api/v1/properties/score-listing` → `{ score: 82, tips: [...] }`

### 7. Two-Factor Authentication (TOTP)
- TOTP (Google Authenticator compatible) — optional per-user
- `POST /api/v1/users/me/2fa/enable` → returns QR code
- `POST /api/v1/auth/verify-totp` → returns full token pair on success
- Backup codes (8 one-time codes) for recovery
- Admin can force-enable 2FA for all agents

### 8. Saved Search Alerts
- Agent saves a market search (area + filters)
- Nightly job checks for new DLD transactions matching criteria
- Notifies agent via WebSocket + email: "3 new sales in Dubai Marina under 2M"

### 9. WhatsApp Template Library
- Admin creates reusable message templates (Arabic + English)
- Templates support variables: `{{client_name}}`, `{{property_area}}`, `{{price}}`
- Agent picks template from dropdown when sending outbound WhatsApp
- Tracks opens/replies per template

### 10. Client Portal (Read-only)
- Shareable link `crm.company.ae/client/[token]` — no login required
- Shows: listings agent has shortlisted for them, comparable prices, area yield
- Client can mark listings as interested / not interested
- Agent notified via WebSocket when client views / acts

---

## Partnership Ideas 🤝

### Pause POS Integration (maidulcu/pause — UAE POS)
Both products are owned by the same developer — tight integration is the goal.

**Direction A — Pause sale → Masaar CRM lead** (highest value, implement first)
- Customer pays at Pause POS terminal
- Pause POSTs to `POST /webhooks/pause/sale` (API key, scope `sale:sync`)
- Masaar upserts contact + creates/updates a lead
- Agent sees new lead in real time via WebSocket

**Direction B — Pause inventory → BOS24 listing** (after push-to-BOS24 feature)
- Unsold/surplus commercial stock in Pause auto-published to BOS24 as a listing
- Needs Roadmap item 5 (push listings to BOS24) completed first

**Pause POS → BOS24 Inventory Sync
- UAE F&B/retail brands on Pause POS have unsold stock
- Auto-sync surplus inventory listings to BOS24's marketplace
- Mutual benefit: Pause gets B2B value-add, BOS24 gets commercial listings

### Masaar CRM → BOS24 (1-click publish)
- See item 5 above — this is the primary integration path
- Masaar becomes "UAE agent OS": CRM + listing publisher in one workspace

### Gemini AI Listing Health Score
- See item 6 above
- Differentiator vs Dubizzle/Bayut: proactive scoring before publish, not after

---

## Known Issues to Fix

| Issue | File | Severity |
|-------|------|----------|
| TOCTOU race in reset token consume | `internal/repo/user.go:ConsumePasswordResetToken` | Medium |
| Non-atomic rate limiter (Incr + Expire) | `internal/api/middleware/api_key.go` | Low |
| `masaar` binary committed to git | `.gitignore` | Low |
| No email format validation on forgot-password | `internal/api/handler/auth.go:ForgotPassword` | Low |
| `aud` claim missing from JWT | `internal/api/handler/auth.go:generateTokenPair` | Low |
| Webhook context 30s may not cover 3 retries | `internal/webhook/dispatcher.go` | Low |
| `c.Locals("timestamp")` always nil in old handlers | `internal/api/handler/property.go` | Fixed in rewrite |
