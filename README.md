<div align="center">

# Masaar CRM — Open-Source WhatsApp CRM for the UAE

**The self-hosted, Arabic-native CRM built for UAE businesses.**

Close deals over WhatsApp · AI thread summaries · Full RTL Arabic UI · PDPL-compliant

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/github/v/release/maidulcu/masaar-crm-community)](https://github.com/maidulcu/masaar-crm-community/releases)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16+-336791?logo=postgresql)](https://postgresql.org)

**Live Demo:** [masaar.dynamicweblab.com](https://masaar.dynamicweblab.com) &nbsp;·&nbsp; Built by [Dynamic Web Lab FZE LLC](https://dynamicweblab.com)

[Features](#features) · [Quick Start](#quick-start) · [Architecture](#architecture) · [API Reference](#api-reference) · [Contributing](#contributing)

</div>

---

## What is Masaar CRM?

Masaar is a **free, open-source CRM designed specifically for UAE businesses**. Most CRM software was built for Western markets with Arabic added as an afterthought. Masaar is different — WhatsApp-first, Arabic-native, and fully self-hosted so your customer data never leaves the UAE.

**Who is it for?**
- UAE SMEs that close deals over WhatsApp
- Sales teams that need a bilingual (Arabic/English) CRM
- Businesses subject to UAE PDPL data residency requirements
- Developers who want a self-hosted alternative to Salesforce, HubSpot, or Zoho

---

## Features

| Feature | Description |
|---------|-------------|
| **Dashboard Overview** | Real-time stats: total contacts, active leads, open WhatsApp threads, pipeline value, won revenue |
| **WhatsApp Inbox** | Receive and manage inbound WhatsApp messages from Meta Cloud API |
| **WhatsApp Outbound** | Send text, media (images/video/documents), and predefined templates via WhatsApp |
| **AI Draft Reply** (Pro) | One-click AI-generated draft replies to WhatsApp threads using local Ollama |
| **AI Buyer Profile** (Pro) | Extract structured buyer requirements from WhatsApp conversations |
| **Sales Pipeline** | Kanban board with drag-and-drop stage management (New → Won/Lost) |
| **Lead Notes** | Click any pipeline card to view and edit per-lead notes inline |
| **AI Thread Summaries** (Pro) | One-click AI summarization via local Ollama LLM — data never leaves your server |
| **Deals & VAT Invoicing** | FTA-style tax invoices with line items, VAT categories, credit notes and a printable bilingual layout; server-side PDF generation is a Pro feature |
| **RERA Compliance** | ORN, BRN and trade licence with expiry alerts; Trakheesi/Madhmoun permits required before a Dubai or Abu Dhabi listing is published or shared |
| **Contact Management** | Unified contact profiles linked to WhatsApp numbers and leads, organisations, custom fields, duplicate detection and merge |
| **Tasks & Follow-ups** | Tasks linked to leads, contacts, deals and listings, with due-date reminders and an overdue view |
| **RERA Forms** | Pre-filled Forms A, B, I and F, printable in English and Arabic, signed through the e-signature link |
| **Property Portals** | Lead webhooks and a listing XML feed per portal (Property Finder, Bayut, Dubizzle, others), lead-source report |
| **Post-dated Cheques** | Cheque schedule per lease, deposit list, bounced-cheque workflow; cleared cheques mark rent paid |
| **Lease Alerts & Notices** | Alerts before leases and Ejari registrations expire and before the rent-change notice deadline; bilingual notices to tenants; rent reminders by email and WhatsApp |
| **Two-factor Authentication** | Authenticator-app codes and backup codes, optional per user or required company-wide |
| **KYC / AML** | Client identity documents, risk and PEP records, beneficial owners, KYC check before a deal is won, AML review queue and goAML export, compliance role |
| **WhatsApp Broadcasts** | Approved templates to consenting segments, STOP/إيقاف opt-out, 24-hour window indicator, conversation assignment, business-hours auto-reply |
| **UAE E-invoicing** | Invoices and credit notes as PINT AE (Peppol) XML, validated against the official rules in CI; provider connector interface with a mock provider and status tracking |
| **Saudi Arabia** | Country rules (15 % VAT, SAR, weekend, regions), REGA licence on listings, optional Hijri dates |
| **Calculators** | Purchase costs (DLD fees), mortgage, rental yield and commission with VAT, shareable on WhatsApp |
| **User Settings** | Per-user password change and language preference (Arabic/English) |
| **Role-Based Access** | Three roles: Admin, Agent, Viewer — enforced on every API route |
| **Real-time Notifications** | Personal WebSocket-powered notifications |
| **Background Jobs** | Viewing, task and rent reminders, offer expiry, licence and lease alerts run automatically; safe with several API replicas |
| **Country Settings** | Per-company country, emirate, currency and timezone; amounts in 3-decimal currencies (BHD, KWD, OMR) are stored exactly |
| **Full RTL Support** | Complete Arabic interface — layout, typography, and date direction |
| **PDPL Tools** | Consent per channel with history, opt-out enforcement, data-subject export and erasure, retention clean-up |
| **Audit Log** | Who created, changed or deleted contacts, leads, deals and documents, for UAE PDPL compliance |
| **Self-Hosted** | Docker Compose (API, web app, PostgreSQL, Redis) — deploy to Moro Hub, G42, or any UAE cloud |

> **Community and Pro.** Features marked *(Pro)* — AI and PDF generation — are stubs in this repository: the endpoints exist but answer with an upgrade message. Market data is also a stub for now (see below). Everything else is fully implemented here. See [CONTRIBUTING.md](CONTRIBUTING.md#whats-in-scope).

---

## Real Estate Market Data

The Properties and Map pages and the property report are built for UAE market data — DLD transactions, Ejari rents, buildings, projects and valuations — from Waqov's [DLD data API](https://dldapi.waqov.com/#api).

**In this edition the market-data client is a stub:** those endpoints answer `503` until a dldapi client is added. The rest of the CRM does not depend on it.

---

## Live Demo

Try it at **[masaar.dynamicweblab.com](https://masaar.dynamicweblab.com)**

> A shared demo: do not enter real customer data.

---

## Quick Start

**Requirements:** Docker and Docker Compose.

```bash
git clone https://github.com/maidulcu/masaar-crm-community.git
cd masaar-crm-community
cp .env.example .env
docker compose up
```

| Service | URL |
|---------|-----|
| Dashboard | http://localhost:3000 |
| API | http://localhost:8080/api/v1 |
| Swagger UI | http://localhost:8080/docs |

**First run:** open http://localhost:3000/signup and create your admin account. Signup closes automatically once the first user exists (`ALLOW_REGISTRATION=false` is the default).

> **Multi-company:** all CRM data (contacts, leads, deals, invoices, WhatsApp, documents, …) is isolated per company. Deployment-wide settings are shared, though: the WhatsApp Cloud API number, SMTP and AI configuration come from environment variables, and inbound WhatsApp messages are routed to the company in `APP_COMPANY_ID` (the first account created on a fresh install). Signup is closed by default (`ALLOW_REGISTRATION=false`); set it to `true` only if you want to host several companies.
>
> In production set `APP_ENV=production` with a real `JWT_SECRET` and `ALLOWED_ORIGINS`; the server refuses to start with placeholder values. `NEXT_PUBLIC_*` variables are baked in at build time — rebuild the web image after changing them.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Your Server (UAE)                     │
│                 Moro Hub / G42 / On-prem                │
│                                                          │
│  ┌──────────┐   ┌──────────┐   ┌──────────────────────┐│
│  │ Next.js  │──▶│  Fiber   │──▶│    PostgreSQL         ││
│  │ (RTL/LTR)│   │  (Go)    │   │                      ││
│  └──────────┘   │          │──▶│    Redis             ││
│                 │          │   └──────────────────────┘│
│  ┌──────────┐   │          │   ┌──────────────────────┐│
│  │ WhatsApp │──▶│ /webhooks│──▶│  Ollama (local LLM)  ││
│  │ Meta API │   │          │   │  llama3 / mistral    ││
│  └──────────┘   └──────────┘   └──────────────────────┘│
│                      │                                   │
│                 ┌────▼─────┐                            │
│                 │  WS Hub  │◀── Real-time Notifications  │
│                 └──────────┘                            │
└─────────────────────────────────────────────────────────┘
         ↑ Docker Compose: API, web app, PostgreSQL, Redis
```

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26 + [Fiber](https://gofiber.io) |
| Frontend | Next.js 14 + Tailwind CSS |
| Database | PostgreSQL 16 + pgvector |
| AI | [Ollama](https://ollama.ai) (llama3 / mistral) |
| Real-time | WebSockets (native Fiber) |
| Cache / Sessions | Redis |
| Migrations | [goose](https://github.com/pressly/goose) |
| Auth | JWT (HS256) + bcrypt |

---

## Project Structure

```
masaar-crm/
├── cmd/server/          # Server entry point
├── internal/
│   ├── api/             # HTTP handlers and middleware
│   ├── domain/          # Models and business types
│   ├── repo/            # PostgreSQL repositories
│   ├── ws/              # WebSocket hub
│   └── ai/              # Ollama integration
├── migrations/          # SQL schema (goose)
├── web/
│   ├── app/             # Next.js App Router pages
│   ├── components/      # React UI components
│   ├── store/           # Zustand state management
│   ├── context/         # Language context (AR/EN)
│   └── lib/             # API client, auth helpers
└── docs/                # OpenAPI / Swagger spec
```

---

## API Reference

All protected routes require `Authorization: Bearer <token>`. Full interactive docs at `/docs`.

| Method | Path | Description | Roles |
|--------|------|-------------|-------|
| `POST` | `/api/v1/auth/login` | Authenticate, get token pair | Public |
| `POST` | `/api/v1/auth/refresh` | Refresh access token | Public |
| `DELETE` | `/api/v1/auth/logout` | Invalidate session | Auth |
| `GET` | `/api/v1/stats` | Dashboard overview metrics | Auth |
| `GET` | `/api/v1/users/me` | Current user profile | Auth |
| `PATCH` | `/api/v1/users/me/password` | Change password | Auth |
| `PATCH` | `/api/v1/users/me/lang` | Update language preference | Auth |
| `GET` | `/api/v1/contacts` | List contacts (search + paginate) | Auth |
| `POST` | `/api/v1/contacts` | Create contact | Agent, Admin |
| `PATCH` | `/api/v1/contacts/:id` | Update contact | Agent, Admin |
| `DELETE` | `/api/v1/contacts/:id` | Delete contact | Admin |
| `GET` | `/api/v1/leads` | Kanban board (all stages) | Auth |
| `POST` | `/api/v1/leads` | Create lead | Agent, Admin |
| `PATCH` | `/api/v1/leads/:id/stage` | Move lead to stage | Agent, Admin |
| `PATCH` | `/api/v1/leads/:id/notes` | Update lead notes | Agent, Admin |
| `GET` | `/api/v1/threads` | List WhatsApp threads | Auth |
| `GET` | `/api/v1/threads/:id` | Get single thread | Auth |
| `GET` | `/api/v1/threads/:id/messages` | Thread messages | Auth |
| `POST` | `/api/v1/threads/:id/close` | Close thread | Agent, Admin |
| `POST` | `/api/v1/threads/:id/send-message` | Send WhatsApp text message | Agent, Admin |
| `POST` | `/api/v1/threads/:id/send-template` | Send WhatsApp template | Agent, Admin |
| `POST` | `/api/v1/threads/:id/send-media` | Send media (image/video/doc) | Agent, Admin |
| `GET` | `/api/v1/threads/:id/outbound-messages` | Sent outbound messages | Agent, Admin |
| `POST` | `/api/v1/ai/draft-reply/:thread_id` | AI-generated draft reply | Agent, Admin |
| `POST` | `/api/v1/ai/extract-buyer-profile/:thread_id` | Extract buyer profile from conversation | Agent, Admin |
| `GET` | `/api/v1/deals` | List deals | Auth |
| `POST` | `/api/v1/deals` | Create deal | Agent, Admin |
| `PATCH` | `/api/v1/deals/:id/stage` | Update deal stage | Agent, Admin |
| `GET` | `/api/v1/invoices/:id` | Get invoice | Auth |
| `POST` | `/api/v1/invoices` | Create VAT invoice | Agent, Admin |
| `POST` | `/api/v1/invoices/:id/send` | Send invoice via WhatsApp | Admin |
| `GET` | `/api/v1/notifications` | List notifications | Auth |
| `PATCH` | `/api/v1/notifications/:id/read` | Mark notification read | Auth |
| `POST` | `/api/v1/ai/summarize/:thread_id` | AI thread summary | Agent, Admin |
| `GET` | `/ws/notifications` | Real-time notification stream | Auth (WS) |
| `GET` | `/api/v1/tasks` | List tasks (filters: mine, status, related record, overdue) | Auth |
| `POST` | `/api/v1/tasks` | Create task | Agent, Admin |
| `GET` | `/api/v1/contacts/duplicates` | Contacts that look like duplicates | Agent, Admin |
| `POST` | `/api/v1/contacts/merge` | Merge contacts | Admin |
| `POST` | `/api/v1/rera-forms` | Create a pre-filled RERA form | Agent, Admin |
| `POST` | `/api/v1/settings/portals/:portal` | Connect a property portal (returns its URLs once) | Admin |
| `POST` | `/webhooks/portals/:portal?token=` | Receive a lead from a property portal | Token |
| `GET` | `/api/public/feeds/:portal/listings.xml?token=` | Listing feed for a portal | Token |
| `GET` | `/api/v1/reports/lead-sources` | Leads, wins and revenue by source | Admin |
| `POST` | `/api/v1/auth/verify-totp` | Complete sign-in with a two-factor code | Pending token |
| `POST` | `/api/v1/users/me/2fa/setup` | Start setting up two-factor | Auth |
| `GET` | `/api/v1/contacts/:id/kyc` | A contact's KYC record | Agent, Admin, Compliance |
| `GET` | `/api/v1/compliance/aml` | AML review queue | Admin, Compliance |
| `POST` | `/api/v1/broadcasts` | Create a WhatsApp broadcast | Admin |
| `GET` | `/api/v1/cheques` | Cheque register (filters: status, from, to, lease) | Auth |
| `POST` | `/api/v1/leases/:id/cheques/schedule` | Create a lease's cheque schedule | Agent, Admin |
| `POST` | `/api/v1/cheques/:id/status` | Deposit, clear, bounce, hold or return a cheque | Agent, Admin |
| `POST` | `/api/v1/leases/:id/notices/draft` | Draft a bilingual notice to the tenant | Agent, Admin |
| `PUT` | `/api/v1/settings/property` | Lease alert days, notice period, rent reminders | Admin |
| | | | |
| **Real Estate Market Data (Dynamic Web Lab)** | *Optional integration; requires API key* | | |
| `POST` | `/api/v1/properties/search` | Search properties (natural language) | Agent, Admin |
| `GET` | `/api/v1/properties/transactions` | List transactions with filters | Agent, Admin |
| `GET` | `/api/v1/properties/buildings` | Search/autocomplete buildings | Agent, Admin |
| `GET` | `/api/v1/properties/buildings/:id` | Building details | Agent, Admin |
| `GET` | `/api/v1/properties/yield-analysis` | Rental yield analysis | Agent, Admin |
| `GET` | `/api/v1/properties/schools/nearby` | Nearby schools & amenities | Agent, Admin |

---

## Contributing

Contributions are welcome. Please open an issue first to discuss significant changes.

```bash
# Fork the repo, then:
git clone https://github.com/maidulcu/masaar-crm-community.git
cd masaar-crm-community
cp .env.example .env
docker compose up -d postgres redis ollama
go run ./cmd/server
```

Please read [CONTRIBUTING.md](CONTRIBUTING.md) before submitting a pull request.

---

## License

Masaar CRM is licensed under the [MIT License](LICENSE). Free to use, modify, and self-host.

---

## Enterprise Edition

Need more for your UAE enterprise? The Enterprise edition adds:

| Feature | Description |
|---------|-------------|
| **Multiple Pipelines** | Sales, HR, Ops — separate pipelines per team |
| **ZATCA E-Invoicing** | Saudi Arabia (ZATCA) e-invoicing with QR code & tax signing |
| **AI Automation** | Auto lead scoring and auto-replies |
| **WhatsApp Bots** | AI-powered chatbots and auto-replies |
| **Advanced Templates** | Multi-language template management and analytics |
| **War Room** | Team live leaderboard dashboard |
| **Semantic Search** | AI-powered similarity search across conversations |
| **SSO / SAML** | Enterprise authentication |
| **Emirates ID** | UAE government ID verification |

**Contact:** [dynamicweblab.com](https://dynamicweblab.com) · info@dynamicweblab.com

---

<div align="center">

Built by [Dynamic Web Lab FZE LLC](https://dynamicweblab.com) &nbsp;·&nbsp; UAE 🇦🇪

**Keywords:** open source CRM UAE · WhatsApp CRM · Arabic CRM software · self-hosted CRM · PDPL compliant CRM · CRM for UAE businesses · Go CRM · bilingual CRM Arabic English

مصنوع بعناية للسوق الإماراتي

</div>
