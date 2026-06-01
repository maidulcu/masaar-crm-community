# Masaar CRM — Project Overview

**Open-source, self-hosted WhatsApp CRM for UAE businesses.**
Built for Arabic-first, WhatsApp-first sales workflows with full property management and real estate integration.

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────┐
│                  Web (Next.js 14)                    │
│  React 18 · Tailwind CSS · Zustand · DnD Kit        │
│  Port 3000                                          │
└──────────────────────┬──────────────────────────────┘
                       │ HTTP/JSON + WebSocket
┌──────────────────────▼──────────────────────────────┐
│              API (Go 1.22+ / Fiber v2)               │
│  180+ REST endpoints · JWT auth · RBAC · Swagger     │
│  Port 8080                                           │
└────┬─────────┬──────────┬──────────┬─────────────────┘
     │         │          │          │
┌────▼──┐ ┌───▼───┐ ┌───▼────┐ ┌───▼──────────────┐
│Postgre│ │ Redis │ │ Ollama │ │ BuyOrSell24 API   │
│SQL 16 │ │   7   │ │ LLM    │ │ (external)        │
│+vector│ │       │ │llama3  │ └───────────────────┘
└───────┘ └───────┘ └────────┘
```

## 2. Tech Stack

| Layer              | Technology                            |
|--------------------|---------------------------------------|
| **Backend**        | Go 1.22+ with Fiber v2 web framework  |
| **Frontend**       | Next.js 14 (App Router) + React 18    |
| **Styling**        | Tailwind CSS 3                        |
| **State**          | Zustand 4                             |
| **Database**       | PostgreSQL 16 + pgvector extension    |
| **Cache/Sessions** | Redis 7                               |
| **AI/LLM**         | Ollama (local) + Gemini (cloud)       |
| **Migrations**     | goose (SQL-based)                     |
| **Auth**           | JWT (HS256) + bcrypt + API keys       |
| **Payments**       | Stripe                                |
| **Real Estate**    | BuyOrSell24 API                       |
| **SMS**            | SMSCountry                            |
| **Email**          | SMTP / Azure Communication Services   |
| **API Docs**       | Swagger/OpenAPI via swaggo            |
| **Real-time**      | WebSocket (Fiber native)              |
| **PDF**            | gofpdf                                |

---

## 3. Project Structure

```
masaar-crm/
├── cmd/server/          # Server entry point (main.go)
├── internal/
│   ├── ai/              # Ollama + Gemini LLM clients (scoring, tagging, summaries)
│   ├── api/
│   │   ├── handler/     # 36 handler files (auth, leads, contacts, deals, etc.)
│   │   ├── middleware/   # JWT auth, RBAC, API keys, rate limiting, quotas, logging
│   │   └── router.go    # 180+ route registrations
│   ├── billing/         # Plan definitions (Community/Starter/Pro/Business) + Stripe
│   ├── bos24/           # BuyOrSell24 real estate API client
│   ├── config/          # Env-based configuration loader
│   ├── domain/          # 95+ domain models (models.go)
│   ├── email/           # SMTP + Azure email service
│   ├── pdf/             # Invoice + property report PDF generation
│   ├── repo/            # 36 repository files (data access layer)
│   ├── sms/             # SMSCountry SMS OTP client
│   ├── webhook/         # Outbound webhook dispatcher (HMAC-signed)
│   ├── whatsapp/        # WhatsApp Cloud API outbound sender
│   └── ws/              # WebSocket hub for real-time notifications
├── web/                 # Next.js frontend
│   ├── app/             # 40+ page routes (App Router)
│   ├── components/      # 16 components (kanban, layout, ui, agent, etc.)
│   ├── lib/             # API client, auth helpers, WebSocket, dates
│   ├── store/           # Zustand auth store
│   ├── context/         # Arabic/English language context
│   ├── hooks/           # useDebounce, useNotifications
│   └── types/           # TypeScript type definitions
├── migrations/          # 40 SQL migration files (goose format)
├── docker/              # Docker Compose files (dev, prod, MVP, server)
├── docs/                # Swagger/OpenAPI generated docs
└── k8s/                 # Kubernetes deployment manifests
```

---

## 4. Domain Models (95+ structs)

### Core CRM
- **User** — name, email, password, role (admin/agent/viewer), language, WhatsApp number, phone, active status
- **Contact** — phone_wa, full_name, email, language, lead_score, assigned_to
- **Lead** — pipeline stage (new→contacted→qualified→proposal→won/lost), source, deal value, currency, score, tags, notes, assigned agent, closed reason, soft delete
- **Deal** — title, stage (open/won/lost), amount, probability, close date, owner
- **VATInvoice** — deal-linked, auto-calculated VAT (5% UAE), invoice number, QR payload, status (draft/sent/paid)
- **WhatsAppThread** — contact, status (open/closed/pending), AI summary, message count
- **WhatsAppMessage** — thread, direction (inbound/outbound), body, media
- **CommunicationHistory** — unified multi-channel log per lead (WhatsApp, email, call)
- **Notification** — user-specific real-time notifications
- **AuditLog** — immutable activity log (entity, action, actor, diff)
- **Stats** — dashboard aggregates (contacts, leads, threads, deals, values)

### Property Management
- **RentalProperty** — type (villa/apartment/townhouse/commercial), address, units, amenities, purchase info, deed
- **Tenant** — full profile: ID docs (Emirati ID/passport/driving license), employment, nationality, emergency contact, verification
- **Lease** — dates, rent, deposit, Ejari number, payment frequency, late fees, signing, termination
- **LeaseTemplate** — reusable lease config (payment terms, deposits, duration, clauses)
- **LeaseRenewalWorkflow** — renewal status, proposed terms, counter-offers, tenant responses
- **Payment** — due/paid dates, method (transfer/check/cash/card), receipts, late fees, reconciliation
- **PaymentReminder** — automated reminders (WhatsApp/email/SMS) with delivery tracking
- **PaymentConfirmation** — confirmation notices with PDF, delivery status
- **Expense** — categories (maintenance/utilities/etc.), vendor, approval workflow, receipts
- **Inspection** — templates with checklists, property inspections with findings, severity, photos
- **MaintenanceTask** — type (plumbing/electrical/HVAC/etc.), priority, contractor, costs, photos
- **Commission** — structures (fixed/percentage/tiered), agent commissions, transactions
- **Document** — templates and generated docs with signatures, audit trail, data classification
- **BankIntegration** — bank account connections with encrypted API credentials
- **BankTransaction** — imported transactions with matching to payments
- **BankStatement** — uploaded files with processing status, transaction import

### Analytics
- **TenantAnalytics** — occupancy, revenue, collection rate, churn
- **PropertyAnalytics** — per-property revenue, expenses, NOI, maintenance
- **FinancialAnalytics** — revenue, expenses, profit, rent collected/pending
- **MaintenanceAnalytics** — task counts, completion rates, average duration

---

## 5. API Routes (180+)

### Auth (9 routes) — public
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/login` | Email + password login with rate limiting and lockout |
| POST | `/api/v1/auth/magic-link/request` | Send passwordless magic link email |
| POST | `/api/v1/auth/magic-link/verify` | Verify magic link token, auto-create account |
| POST | `/api/v1/auth/sms/request` | Request SMS OTP (rate-limited to 3/hr) |
| POST | `/api/v1/auth/sms/verify` | Verify OTP, find-or-create user by phone |
| POST | `/api/v1/auth/refresh` | Rotate refresh token |
| POST | `/api/v1/auth/forgot-password` | Send password reset email |
| POST | `/api/v1/auth/reset-password` | Consume reset token, set new password |
| DELETE | `/api/v1/auth/logout` | Blacklist access + delete refresh token |

### Leads (9 routes) — JWT + RBAC
| Method | Path | Roles | Description |
|--------|------|-------|-------------|
| GET | `/api/v1/leads` | All | Kanban board (grouped by stage) |
| GET | `/api/v1/leads/search` | All | Filtered list with pagination |
| GET | `/api/v1/leads/:id` | All | Single lead with contact |
| POST | `/api/v1/leads` | Admin/Agent | Create lead |
| PATCH | `/api/v1/leads/:id/stage` | Admin/Agent | Move stage (drag-drop) + closed_reason |
| PATCH | `/api/v1/leads/:id/notes` | Admin/Agent | Update notes |
| PATCH | `/api/v1/leads/:id/assign` | Admin/Agent | Assign to agent |
| DELETE | `/api/v1/leads/:id` | Admin | Soft delete |
| GET | `/api/v1/leads/:id/communications` | All | Communication history |

### Contacts (5 routes) — JWT + RBAC
| Method | Path | Roles | Description |
|--------|------|-------|-------------|
| GET | `/api/v1/contacts` | All | List with search + pagination |
| GET | `/api/v1/contacts/:id` | All | Single contact |
| POST | `/api/v1/contacts` | Admin/Agent | Create |
| PATCH | `/api/v1/contacts/:id` | Admin/Agent | Update |
| DELETE | `/api/v1/contacts/:id` | Admin | Delete |

### WhatsApp (6 routes)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/webhooks/whatsapp` | Webhook verification (Meta challenge) |
| POST | `/webhooks/whatsapp` | Receive inbound messages |
| GET | `/api/v1/threads` | List WhatsApp threads |
| GET | `/api/v1/threads/:id` | Single thread with messages |
| GET | `/api/v1/threads/:id/messages` | Thread messages |
| POST | `/api/v1/threads/:id/close` | Close thread |

### Deals (6 routes)
| Method | Path | Roles | Description |
|--------|------|-------|-------------|
| GET | `/api/v1/deals` | All | List with pagination |
| GET | `/api/v1/deals/:id` | All | Single deal |
| POST | `/api/v1/deals` | Admin/Agent | Create |
| PATCH | `/api/v1/deals/:id` | Admin/Agent | Update |
| PATCH | `/api/v1/deals/:id/stage` | Admin/Agent | Move stage |
| GET | `/api/v1/deals/:id/invoices` | All | Deal invoices |

### Invoices (5 routes)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/invoices` | Create with auto invoice number + VAT |
| GET | `/api/v1/invoices/:id` | Get invoice |
| POST | `/api/v1/invoices/:id/send` | Mark as sent |
| PATCH | `/api/v1/invoices/:id/status` | Update status (draft/sent/paid) |
| GET | `/api/v1/invoices/:id/pdf` | Download PDF |

### Real Estate — BOS24 (50+ routes)
Property search, transactions by area, building lookup, area details, rental history, valuation estimates, developer info, project/unit details, map layers, market insights, broker lookup, land records, heatmap, comparables — all via BuyOrSell24 API.

### AI (2 routes)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/ai/summarize/:thread_id` | Summarize WhatsApp thread (Ollama/PII-safe) |
| POST | `/api/v1/ai/describe-listing` | Generate property listing copy (Gemini/non-PII) |

### Messages (3 routes) — AI-powered
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/messages/analyze` | Extract intent from message |
| POST | `/api/v1/messages/suggest-action` | Recommend next action |
| POST | `/api/v1/messages/auto-create-lead` | Auto-create lead from WhatsApp message |

### Property Management
| Entity | Routes | Description |
|--------|--------|-------------|
| Rental Properties | 5 | Full CRUD |
| Tenants | 6 | CRUD + verify identity |
| Lease Templates | 5 | CRUD |
| Leases | 5 | CRUD |
| Payments | 5 | CRUD |
| Lease Renewals | 10 | Initiate, propose, send offer, accept/reject, counter-offer, templates |
| Expenses | 8 | Categories CRUD, expenses CRUD + approve |
| Inspections | 7 | Templates CRUD, inspections CRUD + complete |
| Maintenance | 8 | Tasks CRUD + complete + photos |
| Documents | 13 | Templates CRUD, documents CRUD + signature flow |

### Admin / Settings
| Entity | Routes | Description |
|--------|--------|-------------|
| Users | 9 | CRUD, invite, profile, password, language |
| Settings | 4 | BOS24 + company settings |
| API Keys | 3 | CRUD |
| Webhook Subscriptions | 4 | CRUD + test |
| Billing | 4 | Plan, usage, checkout, portal |
| Bank Integrations | 5 | CRUD |
| Bank Statements | 4 | List, get, upload, delete |

### Analytics (7 routes)
Tenant overview, per-property, per-tenant, financial, maintenance — all with pre-aggregated queries.

### Public / External
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/webhooks/leads` | API Key | External lead intake |
| POST | `/webhooks/stripe` | Stripe signature | Stripe webhook handler |
| GET | `/api/public/sign/:id` | Public | View document for signing |
| POST | `/api/public/sign/:id` | Public | Submit signature |
| GET | `/health` | Public | DB + Redis health check |

---

## 6. Database Migrations (40 files)

| # | Migration | Purpose |
|---|-----------|---------|
| 001 | `create_companies` | Multi-tenant companies table |
| 002 | `create_users` | Auth, roles, language preference |
| 003 | `create_contacts` | WhatsApp contacts with lead score |
| 004 | `create_whatsapp` | Threads + messages with AI summary + vector embeddings |
| 005 | `create_leads` | Pipeline stages, source, deal value (CHECK constraints) |
| 006 | `create_deals` | Deal stages, amounts, probability, owner |
| 007 | `create_invoices` | VAT invoices with auto-calculated columns (5% UAE VAT) |
| 008 | `create_audit_logs` | Immutable audit trail (JSONB diff) |
| 009 | `create_notifications` | User-scoped notifications |
| 010 | `create_api_settings` | Key-value integration settings store |
| 011 | `create_email_history` | Sent email tracking |
| 012 | `create_company_settings` | Company info for invoices (VAT, bank) |
| 013 | `create_whatsapp_outbound` | Outbound message tracking with delivery statuses |
| 014 | `create_lead_tags` | Categorized tags (segment/quality/interest) + lead_score column |
| 015 | `create_communication_history` | Unified multi-channel communication log |
| 016 | `create_rental_properties` | Full property profiles, purchase info, deed |
| 017 | `create_tenants` | ID docs, employment, verification status |
| 018 | `create_lease_templates` | Reusable lease config (terms, deposits, fees) |
| 019 | `create_leases` | Rent schedules, Ejari, signing, termination |
| 020 | `create_bank_integrations` | Encrypted bank API credentials |
| 021 | `create_bank_transactions` | External transaction import |
| 022 | `create_payments` | Payment tracking, reconciliation, late fees |
| 023 | `add_bank_payment_cross_references` | Link bank transactions ↔ payments |
| 024 | `alter_users_default_role` | Change default role to viewer (least-privilege) |
| 025 | `create_bank_statements_and_confirmations` | File uploads + confirmation delivery |
| 026 | `create_inspection_and_maintenance` | Inspections, checklists, maintenance tasks, photos |
| 027 | `create_lease_renewals` | Renewal workflows, counter-offers, communication log |
| 028 | `create_commission_tracking` | Agent commissions (fixed/percentage/tiered) |
| 029 | `create_document_management` | Templates, signature flow, audit trail, data classification |
| 030 | `create_custom_fields` | Extensible entity fields |
| 031 | `create_bulk_operations` | CSV import jobs + error tracking |
| 032 | `create_expenses` | Categories + approval workflow |
| 033 | `create_api_keys` | Scoped API keys with SHA256 hashing |
| 034 | `create_webhooks` | Outbound subscriptions with HMAC signing |
| 036 | `improve_leads` | assigned_to, closed_reason, last_contacted_at, soft delete |
| 037 | `password_reset_tokens` | Token hash, expiry, single-use tracking |
| 038 | `billing_and_plans` | Stripe integration + usage counters |
| 039 | `users_is_active` | Account deactivation support |
| 040 | `users_wa_number_not_null` | Data integrity enforcement |
| 042 | `production_indexes` | Performance indexes on high-traffic tables |
| 043 | `users_phone` | Phone column for SMS OTP login |

---

## 7. Authentication & Authorization

### Auth Methods
1. **Password Login** — email + bcrypt, rate-limited (5 fails = 15-min lockout)
2. **Magic Link** — passwordless email login, token in Redis (single-use)
3. **SMS OTP** — 6-digit code via SMSCountry, rate-limited (3/hr)
4. **JWT Refresh** — rotating refresh tokens in Redis, single-use (replay protection)
5. **API Keys** — `sk_live_` prefixed, SHA256 hashed in DB, scoped per-resource

### Role-Based Access Control (RBAC)
| Permission | Viewer | Agent | Admin |
|------------|--------|-------|-------|
| Read all resources | ✅ | ✅ | ✅ |
| Create leads, contacts, deals | ❌ | ✅ | ✅ |
| Update stages, notes | ❌ | ✅ | ✅ |
| Delete contacts/leads | ❌ | ❌ | ✅ |
| Manage users | ❌ | ❌ | ✅ |
| Manage settings | ❌ | ❌ | ✅ |
| View analytics | ✅ | ✅ | ✅ |

### Middleware Stack
Each protected request passes through:
1. `JWT()` — validates token signature + expiry
2. `ExtractClaims()` — sets user_id, company_id, role in context
3. `CheckBlacklist(rdb)` — verifies token not revoked (fails-closed on Redis error)
4. `RequireRole(admin, agent...)` — enforces minimum role
5. `CheckQuota()` — per-company monthly usage limits
6. `CheckUserAIQuota()` — per-user daily AI rate limits

---

## 8. Key Features

### WhatsApp CRM
- **Inbound**: Meta Cloud API webhook → auto-create contacts/leads → store messages → AI thread summaries
- **Outbound**: Send messages + pre-approved templates via WhatsApp Cloud API
- **Real-time**: WebSocket notifications on new messages, lead updates
- **Thread management**: Open/close threads, AI-powered summaries

### Sales Pipeline (Kanban)
- 6 stages: New → Contacted → Qualified → Proposal → Won → Lost
- Drag-and-drop with optimistic updates
- Lead detail modal with tabs: History, AI Assist, Notes
- Communication history per lead
- AI-assisted scoring and auto-tagging

### Property Management
- Rental properties with full profiles (units, amenities, purchase info)
- Tenants with identity verification and document management
- Lease management with templates, Ejari tracking, renewals
- Payment tracking with reconciliation, reminders, late fees
- Expense management with approval workflow
- Inspections with checklist templates
- Maintenance tasks with contractor assignment and photo tracking

### Real Estate Integration (BuyOrSell24)
- Property search across UAE listings
- Transaction history, valuation estimates
- Area/building/developer/project details
- Heatmaps, comparables, market insights
- Broker and land records

### AI Features
- **Thread summarization** (Ollama, local) — PII-safe, never leaves server
- **Lead scoring** — automated scoring on stage change, message activity, time decay
- **Auto-tagging** — categorize leads from WhatsApp message content
- **Action suggestions** — next-best-action recommendations for agents
- **Property listing copy** (Gemini, cloud) — non-PII marketing content
- **Payment confirmations & reminders** — AI-generated messages

### Billing & Plans
| Plan | Price | BOS24 Quota | AI Quota | PDFs | Key Difference |
|------|-------|-------------|----------|------|----------------|
| Community | Free | 0 | 0 | 0 | Core CRM only |
| Starter | $29/mo | 200 | 100 | 10 | + market data + AI |
| Pro | $99/mo | 2000 | 500 | 100 | + webhooks + API keys |
| Business | $299/mo | 10000 | ∞ | ∞ | + priority support |

---

## 9. Frontend Pages (~40 routes)

### Public Pages
- `/login` — Login with password, magic link, SMS OTP
- `/forgot-password` — Request password reset
- `/reset-password` — Set new password from token
- `/login/magic-link/verify` — Magic link verification
- `/sign/[id]` — Public document signing
- `/developers` — API documentation

### Dashboard (Authenticated)
- `/dashboard` — Overview stats (contacts, leads, threads, deals, quick links)
- `/contacts` — Contact list with search, pagination, add contact
- `/pipeline` — Kanban sales pipeline with drag-and-drop
- `/inbox` — WhatsApp thread list + thread detail
- `/deals` — Deal list + deal detail
- `/properties` — Real estate property search (BOS24)
- `/rentals` — Rental property management
- `/tenants` — Tenant management
- `/leases` — Lease list + lease detail
- `/payments` — Payment management
- `/renewals` — Lease renewal management
- `/inspections` — Inspection list
- `/maintenance` — Maintenance task list
- `/documents` — Document list + template editor
- `/expenses` — Expense management
- `/analytics` — Tenant, property, financial, maintenance analytics
- `/settings` — Company settings, API keys, billing, bank integrations, webhooks
- `/admin/users` — User management (admin only)

---

## 10. Middleware

| Middleware | File | Purpose |
|------------|------|---------|
| `JWT()` | `auth.go` | Validates JWT Bearer token |
| `RequireRole()` | `auth.go` | RBAC enforcement (admin/agent/viewer) |
| `ExtractClaims()` | `auth.go` | Sets user_id, company_id, role in context |
| `CheckBlacklist()` | `auth.go` | Redis-backed token revocation, fail-closed |
| `ValidateAPIKey()` | `api_key.go` | SHA256 API key validation |
| `RequireAPIKeyScope()` | `api_key.go` | Per-key scope enforcement |
| `APIKeyRateLimit()` | `api_key.go` | Per-key Redis sliding window rate limit |
| `PIISafeLogger()` | `logger.go` | Omits query params from logs (phone numbers) |
| `CheckQuota()` | `quota.go` | Per-company monthly usage (Redis fast path) |
| `CheckUserAIQuota()` | `quota.go` | Per-user daily AI limit |

---

## 11. Repository Layer (36 files)

Each domain entity has a corresponding repository in `internal/repo/` that handles:
- SQL queries with pgx prepared statements
- Pagination and filtering where applicable
- Domain struct scanning
- Error wrapping

Notable repos: `LeadRepo` (Kanban, scoring, soft delete), `WhatsAppRepo` (threads, messages, vector embeddings), `AuditLogRepo` (immutable logging), `PaymentRepo` (reconciliation, late fees), `BillingRepo` (usage counters, Stripe sync).

---

## 12. Environment Variables

### Required
| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `JWT_SECRET` | Min 32 random chars for token signing |

### WhatsApp
| Variable | Description |
|----------|-------------|
| `WA_VERIFY_TOKEN` | Webhook verification token |
| `WA_PHONE_NUMBER_ID` | Meta Business phone ID |
| `WA_ACCESS_TOKEN` | Meta API access token |
| `WA_APP_SECRET` | App secret for webhook signature validation |

### AI
| Variable | Default | Description |
|----------|---------|-------------|
| `AI_PROVIDER` | `ollama` | `ollama` or `gemini` |
| `OLLAMA_BASE_URL` | `http://ollama:11434` | Ollama server URL |
| `OLLAMA_MODEL` | `mistral` | Local LLM model |
| `GEMINI_API_KEY` | — | Google Gemini API key |

### Optional Integrations
| Variable | Description |
|----------|-------------|
| `BOS24_API_TOKEN` | BuyOrSell24 real estate data |
| `SMSCOUNTRY_AUTH_KEY` / `AUTH_TOKEN` | SMS OTP login |
| `STRIPE_SECRET_KEY` | Payment processing |
| `SMTP_HOST` / `USER` / `PASSWORD` | Email delivery |

---

## 13. Quick Start

```bash
# Full stack with Docker
cp .env.example .env
docker compose -f docker/docker-compose.yml up

# Backend only (dependencies in Docker)
docker compose -f docker/docker-compose.yml up -d postgres redis ollama
go run ./cmd/server

# Frontend only
cd web && npm install && npm run dev

# Default login: admin@masaar.local / changeme
# API: http://localhost:8080/api/v1
# Swagger: http://localhost:8080/docs
```

---

## 14. Deployment

### Docker (recommended)
Single `docker compose` deployment with:
- PostgreSQL 16 + pgvector
- Redis 7 with AOF persistence
- Ollama for local AI
- Go API binary
- Next.js static export

### Production Requirements
- 2 vCPU, 4GB RAM minimum
- PostgreSQL 16 with pgvector extension
- SSL termination via nginx/Caddy
- Environment-specific `.env` configuration

### Kubernetes
Full K8s manifests available in `k8s/`:
- Deployments, services, ingress for all 5 services
- ConfigMaps + Secrets for environment config
- HPA for auto-scaling API and web
- Pod disruption budgets for HA

---

## 15. File Count

| Area | Files |
|------|-------|
| Backend Go | 55+ source files |
| Frontend TypeScript/TSX | 60+ files |
| SQL Migrations | 40 files |
| Docker/Config | 12+ files |
| Documentation | 20+ markdown files |
| **Total** | **~190+ files** |
