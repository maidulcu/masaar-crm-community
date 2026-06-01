<div align="center">

# Masaar CRM — Open-Source WhatsApp CRM for the UAE

**The self-hosted, Arabic-native CRM built for UAE businesses.**

Close deals over WhatsApp · AI thread summaries · Full RTL Arabic UI · PDPL-compliant

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Version](https://img.shields.io/github/v/release/dynamicweblab/masaar-crm)](https://github.com/dynamicweblab/masaar-crm/releases)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)](https://go.dev)
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
| **AI Draft Reply** | One-click AI-generated draft replies to WhatsApp threads using local Ollama |
| **AI Buyer Profile** | Extract structured buyer requirements from WhatsApp conversations |
| **Sales Pipeline** | Kanban board with drag-and-drop stage management (New → Won/Lost) |
| **Lead Notes** | Click any pipeline card to view and edit per-lead notes inline |
| **AI Thread Summaries** | One-click AI summarization via local Ollama LLM — data never leaves your server |
| **Deals & VAT Invoicing** | Deal tracking with PDF invoice generation including UAE 5% VAT |
| **Contact Management** | Unified contact profiles linked to WhatsApp numbers and leads |
| **User Settings** | Per-user password change and language preference (Arabic/English) |
| **Role-Based Access** | Three roles: Admin, Agent, Viewer — enforced on every API route |
| **Real-time Notifications** | Personal WebSocket-powered notifications |
| **Full RTL Support** | Complete Arabic interface — layout, typography, and date direction |
| **Audit Log** | Immutable activity log for UAE PDPL compliance |
| **Self-Hosted** | Single Docker binary — deploy to Moro Hub, G42, or any UAE cloud |

---

## 🏢 Real Estate Market Data Integration

**Masaar CRM integrates with [BuyOrSell24 by Dynamic Web Lab](https://dynamicweblab.com/products/real-estate-data-api/)** — the UAE's real estate data API — to add market intelligence directly into your CRM workflow.

### What You Get
- **Property Search** — Natural language queries: "2BR apartments in Marina" returns recent transactions, comparable prices, and market trends
- **Transaction History** — Filter by area, property type, price range — see all recent sales/rentals in one click
- **Building Directory** — Quick autocomplete for projects, buildings, and nearby amenities (schools, gyms, metro stations)
- **Yield Analysis** — Investors get rental vs. sales comparison for ROI calculations
- **Market Intelligence** — Price trends, transaction volume, per-sqm analytics

### How to Enable

**Step 1: Get Your API Key**
1. Visit [https://dynamicweblab.com/products/real-estate-data-api/](https://dynamicweblab.com/products/real-estate-data-api/)
2. View pricing tiers and features
3. **Contact us** to set up your account (integration form on landing page)
4. We'll send your API key via Manukaub Bank payment & setup

**Step 2: Configure in Masaar**
```bash
# In your .env file:
BOS24_API_TOKEN=your-api-token-received-from-setup
```

**Step 3: Start Using**
- Agents search properties directly in leads: "Market data for 2BR Dubai Marina"
- Auto-enriched lead cards show comparable sales and price insights
- Investors analyze rental yields in seconds
- All data stays in your self-hosted Masaar instance

**Full API Documentation:** [https://data.buyorsell24.com/redoc](https://data.buyorsell24.com/redoc)

### Pricing & Plans
- **Masaar Pro users** — BuyOrSell24 API credits included in your subscription
- **Open-source users** — Optional integration; contact Dynamic Web Lab for pricing:
  - **Starter** — Limited monthly queries, great for testing
  - **Growth** — 5,000+ monthly credits (perfect for 10-50 agents)
  - **Enterprise** — Custom unlimited credits with dedicated support
  
👉 **[Contact Dynamic Web Lab](https://dynamicweblab.com/products/real-estate-data-api/)** to discuss your requirements and get a quote

> 💡 **For Masaar Pro Users:** Your CRM subscription includes API credits for unlimited property searches. Market intelligence without additional cost!

---

## Live Demo

Try it at **[masaar.dynamicweblab.com](https://masaar.dynamicweblab.com)**

> Resets every 24 hours. Do not enter real customer data.

---

## Quick Start

**Requirements:** Docker and Docker Compose.

```bash
git clone https://github.com/dynamicweblab/masaar-crm.git
cd masaar-crm
cp .env.example .env
docker compose up
```

| Service | URL |
|---------|-----|
| Dashboard | http://localhost:3000 |
| API | http://localhost:8080/api/v1 |
| Swagger UI | http://localhost:8080/docs |

Default login: `admin@masaar.local` / `changeme`

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
         ↑ Single Docker binary — no external dependencies
```

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.22 + [Fiber](https://gofiber.io) |
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
git clone https://github.com/YOUR_USERNAME/masaar-crm.git
cd masaar-crm
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
| **ZATCA E-Invoicing** | UAE FTA compliant with QR code & tax signing |
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
