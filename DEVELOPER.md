# Masaar CRM — Developer Guide

**Version 1.0 | UAE Real Estate CRM | Open Source**

---

## Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Authentication](#authentication)
4. [API Keys (External Integrations)](#api-keys)
5. [API Reference](#api-reference)
6. [Environment Variables](#environment-variables)
7. [Webhooks](#webhooks)
8. [AI Integration](#ai-integration)
9. [Real Estate Data (BOS24)](#real-estate-data)
10. [Database Schema Overview](#database-schema)
11. [Integration Guide](#integration-guide)
12. [Roadmap — What's Coming](#roadmap)

---

## Overview

Masaar CRM is a self-hosted WhatsApp-first CRM for UAE businesses. It exposes a REST API for all CRM operations, a WebSocket endpoint for real-time notifications, and webhook support for Meta's WhatsApp Business API.

**Base URL:** `http://your-host:8080`  
**Interactive Docs:** `http://your-host:8080/docs` (Swagger UI)  
**Health Check:** `GET /health`

---

## Quick Start

```bash
# Clone and configure
git clone https://github.com/maidulcu/masaar-crm
cp .env.example .env   # Fill in your values

# Run with Docker
docker compose up

# Dashboard:   http://localhost:3000
# API:         http://localhost:8080/api/v1
# Swagger UI:  http://localhost:8080/docs
# First run:   create your admin at http://localhost:3000/signup
```

---

## Authentication

### Login (JWT)

All API endpoints require a valid JWT Bearer token, except public webhooks and the login endpoint.

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "you@example.com",
  "password": "<your password>"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "user": { "id": "...", "email": "...", "role": "admin" }
}
```

Use the access token in every subsequent request:
```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

### Token Refresh

Access tokens expire after 15 minutes (configurable). Use the refresh token to get a new one:

```http
POST /api/v1/auth/refresh
Authorization: Bearer <refresh_token>
```

### Token Expiry & Logout

```http
DELETE /api/v1/auth/logout
Authorization: Bearer <access_token>
```

Logged-out tokens are added to a Redis blacklist and immediately rejected.

### Role-Based Access

| Role    | Permissions |
|---------|-------------|
| `admin` | Full access — create/update/delete, manage users and settings |
| `agent` | Create and update leads, contacts, deals; manage communications |
| `viewer`| Read-only access to all resources (default for new users) |

JWT payload includes `role` claim. Role enforcement happens per-endpoint in the API layer.

---

## OpenAPI

The API is described by an **OpenAPI 3.1** document, served without authentication at:

```
GET /openapi.json
```

Import it into Postman, Insomnia, Zapier, Make or an OpenAPI code generator. It covers the documented `/api/v1` routes, the public routes and the lead intake. Session routes use `BearerAuth` (JWT); routes open to API keys also list `ApiKeyAuth` with the scope they need (also in `x-api-key-scope`), document the `Idempotency-Key` header on writes and the RFC 9457 `Problem` error body. `servers` is relative (`/`), so paths resolve against the host you fetched the document from.

The document is generated from the swag annotations on the handlers (`scripts/openapi.sh`) and checked against the router in CI. The Swagger UI at `/docs` shows the same annotations in Swagger 2.0 form (in production only behind `DOCS_PASSWORD`).

## API Keys

For external integrations (website forms, Zapier, automation tools) that can't go through the user login flow, use API keys.

### How It Works

1. Admin generates an API key with specific scopes in Settings
2. The key is returned **once** in plaintext — store it securely
3. Use the key as a Bearer token: `Authorization: Bearer sk_live_...`
4. The API validates the SHA-256 hash against the database

### Scopes

Scope names follow the OAuth 2.0 convention (`resource:action`). A key gets only the scopes it is created with; new keys must use the scopes below (`GET /api/v1/settings/api-keys/scopes` lists them).

| Scope | Allows |
|-------|--------|
| `lead:create` | `POST /webhooks/leads` (lead intake) |
| `leads:read` | `GET /api/v1/leads/search`, `GET /api/v1/leads/{id}` |
| `leads:write` | `POST /api/v1/leads`, `PATCH /api/v1/leads/{id}/stage`, `PATCH /api/v1/leads/{id}/notes`; includes `lead:create` |
| `contacts:read` | `GET /api/v1/contacts`, `GET /api/v1/contacts/{id}` |
| `contacts:write` | `POST /api/v1/contacts`, `PATCH /api/v1/contacts/{id}` |
| `listings:read` | `GET /api/v1/listings`, `GET /api/v1/listings/{id}` |
| `webhooks:manage` | `GET /api/v1/settings/webhooks`, `POST /api/v1/settings/webhooks`, `DELETE /api/v1/settings/webhooks/{id}` |

Every other `/api/v1` route refuses API keys with `403`. Requests and responses are the same as for a signed-in user; data is always limited to the key's company. Each key is limited to 300 requests a minute.

### Calling the API with a key

```bash
curl https://crm.yourcompany.ae/api/v1/leads/search?stage=new \
  -H "Authorization: Bearer sk_live_..."
```

**Errors** to API-key requests are [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) problem details (`Content-Type: application/problem+json`). The `error` field is kept for older clients:

```json
{"type": "about:blank", "title": "Forbidden", "status": 403, "detail": "insufficient_scope", "error": "insufficient_scope", "scope": "leads:write"}
```

A missing scope also sets `WWW-Authenticate: Bearer error="insufficient_scope", scope="leads:write"`; an unknown or revoked key gets `401` with `error="invalid_token"` ([RFC 6750](https://www.rfc-editor.org/rfc/rfc6750#section-3)).

**Safe retries.** Send an `Idempotency-Key` header (1–255 printable ASCII characters, e.g. a UUID) on `POST` and `PATCH`. A retry with the same key and the same request returns the first response with `Idempotent-Replayed: true` instead of acting twice. The same key with a different body gets `422`; a retry while the first request is still running gets `409`. Keys are kept for 24 hours, per API key. A `5xx` response is not stored, so the retry runs again.

### Manage API Keys

```http
# List active keys (admin only)
GET /api/v1/settings/api-keys
Authorization: Bearer <jwt_token>

# Generate new key (admin only)
POST /api/v1/settings/api-keys
Authorization: Bearer <jwt_token>
Content-Type: application/json

{
  "name": "Zapier Integration",
  "scopes": "leads:write,contacts:write"
}

# Response (plaintext shown ONCE — save immediately)
{
  "id": "uuid",
  "name": "Zapier Integration",
  "key_prefix": "sk_live_abcd1234",
  "plaintext": "sk_live_a1b2c3d4e5f6...",
  "scopes": "contacts:write,leads:write"
}

# Revoke key (admin only)
DELETE /api/v1/settings/api-keys/{id}
Authorization: Bearer <jwt_token>
```

---

## API Reference

> Full interactive documentation: `GET /docs`

All authenticated routes are prefixed with `/api/v1`.

### Contacts

```http
GET    /api/v1/contacts          # List all contacts (all roles)
GET    /api/v1/contacts/:id      # Get contact (all roles)
POST   /api/v1/contacts          # Create contact (agent+)
PATCH  /api/v1/contacts/:id      # Update contact (agent+)
DELETE /api/v1/contacts/:id      # Delete contact (admin)
```

**Create Contact body:**
```json
{
  "full_name": "Ahmed Al-Mansouri",
  "phone_wa": "+971501234567",
  "email": "ahmed@example.com",
  "language": "ar"
}
```
> Phone must be E.164 format: `+[country code][number]`

### Leads

```http
GET    /api/v1/leads              # Kanban board (all stages, all roles)
GET    /api/v1/leads/:id          # Get lead (all roles)
POST   /api/v1/leads              # Create lead (agent+)
PATCH  /api/v1/leads/:id/stage    # Move stage (agent+)
PATCH  /api/v1/leads/:id/notes    # Update notes (agent+)
GET    /api/v1/leads/:id/communications  # Communication history
```

**Create Lead body:**
```json
{
  "contact_id": "uuid",
  "stage": "new",
  "source": "web",
  "deal_value": 500000,
  "currency": "AED",
  "notes": "Interested in Marina 2BR"
}
```

**Lead stages:** `new` → `contacted` → `qualified` → `proposal` → `won` / `lost`  
**Lead sources:** `whatsapp`, `web`, `referral`, `event`

### WhatsApp Threads

```http
GET  /api/v1/threads                          # List threads
GET  /api/v1/threads/:id                      # Get thread
GET  /api/v1/threads/:id/messages             # Messages in thread
POST /api/v1/threads/:id/close                # Close thread (agent+)
POST /api/v1/threads/:id/send-message         # Send message (agent+)
POST /api/v1/threads/:id/send-template        # Send template (agent+)
GET  /api/v1/threads/:id/outbound-messages    # Sent messages history
```

### Deals

```http
GET   /api/v1/deals                # List deals
POST  /api/v1/deals                # Create deal (agent+)
PATCH /api/v1/deals/:id/stage      # Update stage (agent+)
GET   /api/v1/deals/:id/invoices   # Deal invoices
```

### Invoices

```http
GET  /api/v1/invoices/:id          # Get invoice
GET  /api/v1/invoices/:id/pdf      # Download PDF
POST /api/v1/invoices              # Create invoice (agent+)
POST /api/v1/invoices/:id/send     # Send invoice (admin)
PATCH /api/v1/invoices/:id/status  # Update status (admin)
```

### Property Management

```http
# Rental Properties
GET    /api/v1/rental-properties
POST   /api/v1/rental-properties        # agent+
PATCH  /api/v1/rental-properties/:id    # agent+
DELETE /api/v1/rental-properties/:id    # admin

# Tenants
GET    /api/v1/tenants
POST   /api/v1/tenants                  # agent+
PATCH  /api/v1/tenants/:id              # agent+
POST   /api/v1/tenants/:id/verify       # admin

# Leases
GET    /api/v1/leases
POST   /api/v1/leases                   # agent+
PATCH  /api/v1/leases/:id

# Payments
GET    /api/v1/payments
POST   /api/v1/payments                 # agent+
```

### Analytics

```http
GET /api/v1/analytics/tenant-overview
GET /api/v1/analytics/financial
GET /api/v1/analytics/properties
GET /api/v1/analytics/tenants
GET /api/v1/analytics/maintenance
```

### AI

```http
POST /api/v1/ai/summarize/:thread_id    # Summarize WhatsApp thread (agent+)
POST /api/v1/messages/analyze           # Analyze message intent (agent+)
POST /api/v1/messages/suggest-action    # Suggest next action (agent+)
POST /api/v1/messages/auto-create-lead  # Auto-create lead from message (agent+)
```

### Settings (Admin Only)

```http
GET   /api/v1/settings/company
PATCH /api/v1/settings/company
GET   /api/v1/settings/bos24
PATCH /api/v1/settings/bos24
GET   /api/v1/settings/api-keys
POST  /api/v1/settings/api-keys
DELETE /api/v1/settings/api-keys/:id
```

### Real Estate Market Data (Waqov)

```http
POST /api/v1/properties/search             # NL search ("2BR in Marina")
GET  /api/v1/properties/transactions       # Transaction history
GET  /api/v1/properties/buildings          # Building search
GET  /api/v1/properties/buildings/:id      # Building details
GET  /api/v1/properties/schools/nearby     # Nearby schools/amenities
GET  /api/v1/properties/yield-analysis     # Rental yield analysis
GET  /api/v1/properties/comparables        # Comparable properties
GET  /api/v1/properties/market-trends      # Market trends
```
> Requires `BOS24_API_TOKEN` configured. See [Waqov DLD data API](https://dldapi.waqov.com/#api).

### WebSocket — Real-Time Notifications

```
GET /ws/notifications
Upgrade: websocket
Authorization: Bearer <jwt_token>
```

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string |
| `REDIS_URL` | ✅ | — | Redis connection string |
| `JWT_SECRET` | ✅ | — | **Must be 32+ random chars in production** |
| `JWT_ACCESS_EXPIRY_MIN` | — | `15` | Access token lifetime (minutes) |
| `JWT_REFRESH_EXPIRY_DAYS` | — | `7` | Refresh token lifetime (days) |
| `ALLOWED_ORIGINS` | ✅ | `*` | CORS — set to your frontend URL in production |
| `APP_COMPANY_ID` | — | `00000000-0000-0000-0000-000000000001` | Single-tenant company UUID |
| `WA_VERIFY_TOKEN` | WhatsApp | — | Meta webhook verification token |
| `WA_APP_SECRET` | WhatsApp | — | App secret for HMAC signature validation |
| `WA_PHONE_NUMBER_ID` | WhatsApp outbound | — | Meta phone number ID |
| `WA_ACCESS_TOKEN` | WhatsApp outbound | — | Meta Business API token |
| `AI_PROVIDER` | — | `ollama` | `ollama` or `gemini` |
| `OLLAMA_BASE_URL` | AI (Ollama) | `http://ollama:11434` | Ollama endpoint |
| `OLLAMA_MODEL` | — | `llama3` | Ollama model name |
| `GEMINI_API_KEY` | AI (Gemini) | — | Google Gemini API key |
| `GEMINI_MODEL` | — | `gemini-2.0-flash` | Gemini model name |
| `BOS24_API_TOKEN` | Real estate | — | Waqov data API token |
| `SMTP_HOST` | Email | — | SMTP server host |
| `SMTP_PORT` | — | `587` | SMTP port |
| `SMTP_USER` | Email | — | SMTP username |
| `SMTP_PASSWORD` | Email | — | SMTP password |
| `SMTP_FROM_EMAIL` | — | `noreply@masaar.local` | Sender address |

---

## Webhooks

### Inbound — WhatsApp (Meta Cloud API)

```
POST /webhooks/whatsapp
X-Hub-Signature-256: sha256=<hmac>
```

Set your webhook URL in Meta Developer Portal to:
```
https://your-domain.com/webhooks/whatsapp
```

**Verification:** On first setup, Meta sends a GET request. The CRM responds automatically using `WA_VERIFY_TOKEN`.

**Security:** Set `WA_APP_SECRET` to your Meta App Secret. The CRM validates every inbound request using HMAC-SHA256.

**Rate limit:** 300 requests/minute per IP.

### Outbound Webhooks

Register a URL and Masaar CRM will push signed events to it automatically:

| Event | Trigger |
|-------|---------|
| `lead.created` | New lead added (via UI or API) |
| `lead.stage_changed` | Lead moves in pipeline |
| `lead.won` | Lead marked won |
| `lead.lost` | Lead marked lost |
| `webhook.test` | Test ping from `POST /api/v1/settings/webhooks/:id/test` |

`payment.received`, `lease.signed` and `contact.created` are reserved names: nothing sends them yet. The events and their `data` are also described in the `webhooks` section of [`/openapi.json`](#openapi).

**Setup via API (Admin only):**

```bash
# Register a webhook
curl -X POST https://crm.yourcompany.ae/api/v1/settings/webhooks \
  -H "Authorization: Bearer JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"My App","url":"https://myapp.com/crm-events","events":"lead.created,lead.won"}'
# Returns: {"id":"...","secret":"whsec_..."}  ← store secret, shown once

# List registered webhooks
GET /api/v1/settings/webhooks

# Send a test ping
POST /api/v1/settings/webhooks/:id/test

# Remove a webhook
DELETE /api/v1/settings/webhooks/:id
```

**Payload envelope:**

```json
{
  "type": "lead.created",
  "event": "lead.created",
  "timestamp": "2025-01-15T10:30:00Z",
  "data": {
    "lead_id": "uuid",
    "contact_id": "uuid",
    "stage": "new",
    "source": "web",
    "deal_value": 500000
  }
}
```

`type` and `event` carry the same value (`event` is kept for older receivers).

**Signature verification ([Standard Webhooks](https://www.standardwebhooks.com)):**

Every request carries three headers:

| Header | Value |
|--------|-------|
| `webhook-id` | Message id (`msg_…`), the same on every retry: use it to skip duplicates |
| `webhook-timestamp` | Unix seconds when this attempt was signed: reject old ones (e.g. older than 5 minutes) |
| `webhook-signature` | `v1,` + base64 HMAC-SHA256 of `webhook-id + "." + webhook-timestamp + "." + raw body` |

The HMAC key is the subscription secret without its `whsec_` prefix, base64-decoded. Any Standard Webhooks library does this for you; pass it the secret exactly as Masaar showed it (secrets created before `whsec_` work unchanged):

```python
from standardwebhooks.webhooks import Webhook   # pip install standardwebhooks

payload = Webhook(secret).verify(raw_body, request.headers)  # raises on a bad signature or old timestamp
```

```javascript
import { Webhook } from 'standardwebhooks';   // npm install standardwebhooks

const payload = new Webhook(secret).verify(rawBody, req.headers);
```

Verify the raw request body, before any JSON parsing.

**Legacy header.** Requests also still carry `X-Masaar-Signature: sha256=<hex HMAC-SHA256 of the body, keyed with the secret string>`, plus `X-Masaar-Event` and `X-Masaar-Delivery` (now the same as `webhook-id`). Existing receivers keep working; new ones should use the headers above.

Delivery retries: 3 attempts with 1s / 4s backoff. Failures tracked in `webhook_deliveries` table.

---

## AI Integration

Masaar CRM supports two AI providers. Configure via `AI_PROVIDER` env var.

### Ollama (Local — Default)

```bash
AI_PROVIDER=ollama
OLLAMA_BASE_URL=http://ollama:11434
OLLAMA_MODEL=llama3    # or mistral, phi3, etc.
```

Runs fully offline. First run downloads ~5GB model.

### Google Gemini (Cloud)

```bash
AI_PROVIDER=gemini
GEMINI_API_KEY=your_key_here
GEMINI_MODEL=gemini-2.0-flash
```

Get API key: https://aistudio.google.com/app/apikey

### AI Capabilities

| Feature | Endpoint | Description |
|---------|----------|-------------|
| Thread Summary | `POST /ai/summarize/:thread_id` | 50-message WhatsApp conversation summary |
| Lead Scoring | Auto | 0–100 score based on stage, recency, engagement |
| Intent Parsing | `POST /messages/analyze` | Detect property type, area, budget from message |
| Lead Enrichment | Auto | Extract structured data from conversation |
| Draft Reply | `POST /ai/score-lead/:id` | Generate WhatsApp reply suggestion |
| Action Suggestion | `POST /messages/suggest-action` | Recommend next agent action |

---

## Real Estate Data

The market-data integration ([Waqov](https://waqov.com/)) provides UAE market data (DLD transactions, Ejari rents, buildings, projects, valuations, maps) from the [Waqov DLD data API](https://dldapi.waqov.com/openapi.json). Optional — all `/api/v1/properties/*` endpoints return 503 if not configured. GET responses are cached in Redis for 10 minutes (listings, transactions) to 24 hours (map config); each call counts against the company's market-data quota.

**Setup:**
1. Get an API token from [Waqov](https://waqov.com/)
2. Set `BOS24_API_TOKEN` in `.env` and restart the API (`BOS24_BASE_URL` defaults to `https://dldapi.waqov.com`). The token saved through `PATCH /api/v1/settings/bos24` is not read by the client yet.

Upstream errors map to `404` (not found), `429` (rate limited) and `503` (anything else, including a rejected key).

**API Docs:** https://dldapi.waqov.com/#api · spec: https://dldapi.waqov.com/openapi.json

---

## Database Schema

Migrations run automatically on startup via [goose](https://github.com/pressly/goose).

| Migration | Table(s) |
|-----------|---------|
| 00000 | `companies` |
| 00001 | `users` |
| 00002 | `contacts` |
| 00003 | `whatsapp_threads`, `whatsapp_messages` |
| 00004 | `leads` |
| 00005 | `deals` |
| 00006 | `invoices` |
| 00007 | `audit_logs` |
| 00008 | `notifications` |
| 00009 | `api_settings` |
| 00010 | `email_history` |
| 00011 | `company_settings` |
| 00012 | `whatsapp_outbound` |
| 00013 | `lead_tags` |
| 00014 | `communication_history` |
| 00015 | `rental_properties` |
| 00016 | `tenants` |
| 00017 | `lease_templates` |
| 00018 | `leases` |
| 00019 | `payments` |
| 00020 | `bank_transactions` |
| 00021 | `bank_integrations` |
| 00022 | `payment_reminders` |
| 00023 | `bank_statements`, `payment_confirmations` |
| 00025 | `inspection_templates`, `inspections`, `maintenance_tasks` |
| 00026 | `lease_renewals` |
| 00027 | `commission_tracking` |
| 00028 | `document_management` |
| 00029 | `custom_fields` |
| 00030 | `expenses`, `bulk_operations` |
| 00031 | `api_keys` |
| 00032 | `webhook_subscriptions`, `webhook_deliveries` |

---

## Integration Guide

### Connecting a Website Form

Use the public lead endpoint with an API key — no user session required:

```bash
# 1. Create an API key (admin, once)
curl -X POST https://crm.yourcompany.ae/api/v1/settings/api-keys \
  -H "Authorization: Bearer JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Website Form","scopes":"lead:create"}'
# Returns: {"plaintext":"sk_live_...", ...}  ← store this once

# 2. Submit leads from your form
curl -X POST https://crm.yourcompany.ae/webhooks/leads \
  -H "Authorization: Bearer sk_live_..." \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ahmed Al-Mansouri",
    "phone": "+971501234567",
    "email": "ahmed@example.com",
    "language": "ar",
    "source": "web",
    "notes": "Interested in Marina apartments",
    "deal_value": 500000,
    "property_type": "2BR",
    "area": "Marina"
  }'
```

### Connecting Zapier / Make.com

Use the REST API module in Zapier/Make with:
- **Auth type:** Bearer Token (use your API key)
- **Base URL:** `https://crm.yourcompany.ae/api/v1`
- **Create Lead action:** `POST /contacts`, then `POST /leads` with the contact's `id` (scopes `contacts:write`, `leads:write`)
- Set an `Idempotency-Key` header so Zapier's automatic retries do not create duplicates

### Connecting from Another CRM (HubSpot, Salesforce)

Use the API directly with a scoped API key. Typical sync flow:

```
External CRM → webhook trigger → your middleware → POST /api/v1/contacts + POST /api/v1/leads
```

---

## Roadmap — What's Missing

### Phase 2 — Public Lead Endpoint ✅ Live

Submit leads from any external source using an API key:

```http
POST /webhooks/leads
Authorization: Bearer sk_live_...
Content-Type: application/json

{
  "name": "Ahmed Al-Mansouri",
  "phone": "+971501234567",
  "email": "ahmed@example.com",
  "language": "ar",
  "source": "web",
  "notes": "Interested in Marina apartments",
  "deal_value": 500000,
  "property_type": "2BR",
  "area": "Marina",
  "external_source": "myportal",
  "external_ref": "enquiry:48213"
}
```

- Requires API key with scope `lead:create`; a suspended company's key gets `403 account_suspended`
- `source` is one of `web`, `referral`, `event`, `whatsapp`, `property_finder`, `bayut`, `dubizzle`, `portal` (default `web`); anything else is `422`
- `external_source` + `external_ref` (optional, sent together; `portal` requires them) make the push idempotent: the same pair again returns the existing lead with `200` and `"duplicate": true` instead of creating a second one. Unique per company
- Contact auto-created or matched by phone number
- Rate limited at 300 req/min (same as WhatsApp webhook)
- Returns `lead_id`, `contact_id`, `stage`, `source`, `external_source`, `external_ref`, `duplicate`
- Each lead created is written to the audit log with the API key as the actor

### Phase 3 — Outbound Webhooks ✅ Live

Register URLs to receive signed events when CRM activity happens. See [Outbound Webhooks](#outbound-webhooks) above for full setup.

Quick start:
```bash
POST /api/v1/settings/webhooks
{"name":"Zapier","url":"https://hooks.zapier.com/...","events":"lead.created,lead.won"}
```

### Phase 4 — OAuth 2.0 *(future)*

For Zapier/Make.com native integrations with OAuth flow.

---

## Contributing

See [CLAUDE.md](./CLAUDE.md) for codebase conventions, handler patterns, and repository architecture.

**Key files:**
- `internal/api/router.go` — All route registrations
- `internal/api/handler/` — HTTP handlers
- `internal/repo/` — Database layer
- `internal/domain/models.go` — Core domain types
- `cmd/server/main.go` — Server wiring

**Adding a new endpoint:**
1. Add migration if needed (`migrations/000XX_description.sql`)
2. Add domain model to `internal/domain/models.go`
3. Add repo methods to `internal/repo/`
4. Create handler in `internal/api/handler/`
5. Register route in `internal/api/router.go`

---

*Masaar CRM is open source under MIT License. Built for UAE businesses.*  
*Issues & contributions: https://github.com/maidulcu/masaar-crm*
