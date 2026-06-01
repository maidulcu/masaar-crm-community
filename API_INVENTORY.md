# API Endpoint Inventory

> Source of truth: `internal/api/router.go` (backend) ↔ `web/lib/api.ts` (frontend wrappers)

**Status Legend:**
- ✅ **Connected** — wrapper exists in `api.ts` AND is used by at least one page/component
- 🔌 **Wrapper only** — wrapper exists in `api.ts` but no page imports/calls it yet
- ❌ **Missing** — no wrapper in `api.ts` at all
- ⚠️ **Wrong URL** — page calls the endpoint with the wrong path
- 🚫 **Webhook/System** — external-only, no frontend needed

---

## Authentication (`/api/v1/auth`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 1 | POST | `/api/v1/auth/login` | public | `api.auth.login()` | `login/page.tsx` | ✅ |
| 2 | POST | `/api/v1/auth/magic-link/request` | public | `api.auth.requestMagicLink()` | `login/page.tsx` | ✅ |
| 3 | POST | `/api/v1/auth/magic-link/verify` | public | `api.auth.verifyMagicLink()` | `login/magic-link/verify/page.tsx` | ✅ |
| 4 | POST | `/api/v1/auth/sms/request` | public | `api.auth.requestSMSOTP()` | `login/page.tsx` | ✅ |
| 5 | POST | `/api/v1/auth/sms/verify` | public | `api.auth.verifySMSOTP()` | `login/page.tsx` | ✅ |
| 6 | POST | `/api/v1/auth/refresh` | public | `api.auth.refresh()` | none | 🔌 |
| 7 | POST | `/api/v1/auth/forgot-password` | public | `api.auth.forgotPassword()` | `forgot-password/page.tsx` | ✅ |
| 8 | POST | `/api/v1/auth/reset-password` | public | `api.auth.resetPassword()` | `reset-password/page.tsx` | ✅ |
| 9 | DELETE | `/api/v1/auth/logout` | any auth | `api.auth.logout()` | `components/layout/Header.tsx` | ✅ |

**Gaps:** `refresh` has a wrapper but no UI calls it. Token refresh likely handled differently (interceptor).

---

## Dashboard & Stats

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 10 | GET | `/api/v1/stats` | any auth | `api.stats.overview()` | `(dashboard)/dashboard/page.tsx` | ✅ |

---

## Users (`/api/v1/users`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 11 | GET | `/api/v1/users` | Admin | `api.users.list()` | `(dashboard)/admin/users/page.tsx` | ✅ |
| 12 | POST | `/api/v1/users` | Admin | none | none | ❌ |
| 13 | POST | `/api/v1/users/invite` | Admin | `api.users.invite()` | `(dashboard)/admin/users/page.tsx` | ✅ |
| 14 | GET | `/api/v1/users/me` | any auth | `api.users.me()` | auth store | ✅ |
| 15 | PATCH | `/api/v1/users/me/password` | any auth | `api.users.changePassword()` | `(dashboard)/settings/page.tsx` | ✅ |
| 16 | PATCH | `/api/v1/users/me/lang` | any auth | `api.users.updateLang()` | `(dashboard)/settings/page.tsx` | ✅ |
| 17 | PATCH | `/api/v1/users/:id` | Admin | `api.users.update()` | `(dashboard)/admin/users/page.tsx` | ✅ |
| 18 | PATCH | `/api/v1/users/:id/active` | Admin | `api.users.setActive()` | `(dashboard)/admin/users/page.tsx` | ✅ |
| 19 | DELETE | `/api/v1/users/:id` | Admin | `api.users.delete()` | `(dashboard)/admin/users/page.tsx` | ✅ |

**Gaps:** `POST /api/v1/users` (direct create, no invite) — admin-only, used only via invite flow.

---

## Settings (`/api/v1/settings`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 20 | GET | `/api/v1/settings/bos24` | Admin | `api.settings.getBOS24()` | `settings/api/page.tsx` (direct fetch) | ✅ |
| 21 | PATCH | `/api/v1/settings/bos24` | Admin | `api.settings.updateBOS24()` | `settings/api/page.tsx` (direct fetch) | ✅ |
| 22 | GET | `/api/v1/settings/company` | Admin | `api.settings.getCompany()` | none | 🔌 |
| 23 | PATCH | `/api/v1/settings/company` | Admin | `api.settings.updateCompany()` | none | 🔌 |
| 24 | GET | `/api/v1/settings/api-keys` | Admin | `api.settings.apiKeys.list()` | none | 🔌 |
| 25 | POST | `/api/v1/settings/api-keys` | Admin | `api.settings.apiKeys.create()` | none | 🔌 |
| 26 | DELETE | `/api/v1/settings/api-keys/:id` | Admin | `api.settings.apiKeys.revoke()` | none | 🔌 |
| 27 | GET | `/api/v1/settings/webhooks` | Admin | `api.settings.webhooks.list()` | none | 🔌 |
| 28 | POST | `/api/v1/settings/webhooks` | Admin | `api.settings.webhooks.create()` | none | 🔌 |
| 29 | DELETE | `/api/v1/settings/webhooks/:id` | Admin | `api.settings.webhooks.delete()` | none | 🔌 |
| 30 | POST | `/api/v1/settings/webhooks/:id/test` | Admin | `api.settings.webhooks.test()` | none | 🔌 |

**Gaps:** Company settings, API keys, and webhooks have wrappers but no pages use them yet. BOS24 settings called via direct fetch.

---

## Billing (`/api/v1/billing`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 31 | GET | `/api/v1/billing` | any auth | `api.billing.get()` | `settings/billing/page.tsx` | ✅ |
| 32 | GET | `/api/v1/billing/usage` | any auth | `api.billing.getUsage()` | none | 🔌 |
| 33 | POST | `/api/v1/billing/checkout` | Admin | `api.billing.checkout()` | `settings/billing/page.tsx` | ✅ |
| 34 | POST | `/api/v1/billing/portal` | Admin | `api.billing.portal()` | `settings/billing/page.tsx` | ✅ |

**Gaps:** `usage` has wrapper but unused. Billing page was fixed from wrong URL paths.

---

## Contacts (`/api/v1/contacts`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 35 | GET | `/api/v1/contacts` | any auth | `api.contacts.list()` | `(dashboard)/contacts/page.tsx`, pipeline | ✅ |
| 36 | GET | `/api/v1/contacts/:id` | any auth | `api.contacts.get()` | none | 🔌 |
| 37 | POST | `/api/v1/contacts` | Admin/Agent | `api.contacts.create()` | `(dashboard)/contacts/page.tsx` | ✅ |
| 38 | PATCH | `/api/v1/contacts/:id` | Admin/Agent | `api.contacts.update()` | none | 🔌 |
| 39 | DELETE | `/api/v1/contacts/:id` | Admin | `api.contacts.delete()` | none | 🔌 |

---

## Leads / Pipeline (`/api/v1/leads`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 40 | GET | `/api/v1/leads` | any auth | `api.leads.kanban()` | `(dashboard)/pipeline/page.tsx` | ✅ |
| 41 | GET | `/api/v1/leads/search` | any auth | `api.leads.search()` | none | 🔌 |
| 42 | GET | `/api/v1/leads/:id` | any auth | `api.leads.get()` | none | 🔌 |
| 43 | POST | `/api/v1/leads` | Admin/Agent | `api.leads.create()` | `(dashboard)/pipeline/page.tsx` | ✅ |
| 44 | PATCH | `/api/v1/leads/:id/stage` | Admin/Agent | `api.leads.updateStage()` | `(dashboard)/pipeline/page.tsx` | ✅ |
| 45 | PATCH | `/api/v1/leads/:id/notes` | Admin/Agent | `api.leads.updateNotes()` | `(dashboard)/pipeline/page.tsx` | ✅ |
| 46 | PATCH | `/api/v1/leads/:id/assign` | Admin/Agent | `api.leads.assign()` | none | 🔌 |
| 47 | DELETE | `/api/v1/leads/:id` | Admin | `api.leads.delete()` | none | 🔌 |
| 48 | GET | `/api/v1/leads/:id/communications` | any auth | `api.leads.communications()` | `(dashboard)/pipeline/page.tsx` | ✅ |

---

## WhatsApp / Threads (`/api/v1/threads`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 49 | GET | `/api/v1/threads` | any auth | `api.threads.list()` | `(dashboard)/inbox/page.tsx`, pipeline | ✅ |
| 50 | GET | `/api/v1/threads/:id` | any auth | `api.threads.get()` | `inbox/[id]/page.tsx` | ✅ |
| 51 | GET | `/api/v1/threads/:id/messages` | any auth | `api.threads.messages()` | `inbox/[id]/page.tsx`, pipeline | ✅ |
| 52 | POST | `/api/v1/threads/:id/close` | Admin/Agent | `api.threads.close()` | `inbox/[id]/page.tsx` | ✅ |
| 53 | POST | `/api/v1/threads/:id/send-message` | Admin/Agent | `api.whatsapp.sendMessage()` | none | 🔌 |
| 54 | POST | `/api/v1/threads/:id/send-template` | Admin/Agent | `api.whatsapp.sendTemplate()` | none | 🔌 |
| 55 | GET | `/api/v1/threads/:id/outbound-messages` | Admin/Agent | `api.whatsapp.getOutboundMessages()` | none | 🔌 |

---

## AI / LLM (`/api/v1/ai`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 56 | POST | `/api/v1/ai/summarize/:thread_id` | Admin/Agent | `api.ai.summarize()` | `inbox/[id]/page.tsx` | ✅ |
| 57 | POST | `/api/v1/ai/extract-buyer-profile/:thread_id` | Admin/Agent | `api.ai.extractBuyerProfile()` | `components/agent/BuyerProfile.tsx` | ✅ |
| 58 | POST | `/api/v1/ai/score-lead/:id` | Admin/Agent | `api.ai.scoreLead()` | none | 🔌 |
| 59 | POST | `/api/v1/ai/draft-reply/:thread_id` | Admin/Agent | `api.ai.draftReply()` | none | 🔌 |
| 60 | POST | `/api/v1/ai/describe-listing` | Admin/Agent | `api.ai.describeListing()` | none | 🔌 |

---

## Messages (`/api/v1/messages`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 61 | POST | `/api/v1/messages/analyze` | Admin/Agent | `api.messages.analyze()` | none | 🔌 |
| 62 | POST | `/api/v1/messages/suggest-action` | Admin/Agent | `api.messages.suggestAction()` | `components/agent/AgentAssist.tsx` | ✅ |
| 63 | POST | `/api/v1/messages/auto-create-lead` | Admin/Agent | `api.messages.autoCreateLead()` | none | 🔌 |

---

## Market Data / Properties (`/api/v1/properties`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 64 | POST | `/api/v1/properties/search` | Admin/Agent | none (direct fetch) | `properties/page.tsx` | ✅ |
| 65 | POST | `/api/v1/properties/projects/search` | Admin/Agent | none | none | ❌ |
| 66 | POST | `/api/v1/properties/ai/describe` | Admin/Agent | none | none | ❌ |
| 67 | GET | `/api/v1/properties/transactions` | Admin/Agent | none (direct fetch) | `properties/page.tsx` | ✅ |
| 68 | GET | `/api/v1/properties/transactions/areas` | Admin/Agent | none | none | ❌ |
| 69 | GET | `/api/v1/properties/buildings` | Admin/Agent | none | none | ❌ |
| 70 | GET | `/api/v1/properties/buildings/:id` | Admin/Agent | none | none | ❌ |
| 71 | GET | `/api/v1/properties/areas` | Admin/Agent | none | none | ❌ |
| 72 | GET | `/api/v1/properties/areas/:slug/summary` | Admin/Agent | none | none | ❌ |
| 73 | GET | `/api/v1/properties/areas/:slug/buildings` | Admin/Agent | none | none | ❌ |
| 74 | GET | `/api/v1/properties/map/areas` | Admin/Agent | none | none | ❌ |
| 75 | GET | `/api/v1/properties/pois` | Admin/Agent | none | none | ❌ |
| 76 | GET | `/api/v1/properties/schools/nearby` | Admin/Agent | none | none | ❌ |
| 77 | GET | `/api/v1/properties/rentals` | Admin/Agent | none | none | ❌ |
| 78 | GET | `/api/v1/properties/rentals/ejari` | Admin/Agent | none | none | ❌ |
| 79 | GET | `/api/v1/properties/rentals/ejari/yield` | Admin/Agent | none | none | ❌ |
| 80 | GET | `/api/v1/properties/developers` | Admin/Agent | none | none | ❌ |
| 81 | GET | `/api/v1/properties/projects` | Admin/Agent | none | none | ❌ |
| 82 | GET | `/api/v1/properties/units` | Admin/Agent | none | none | ❌ |
| 83 | GET | `/api/v1/properties/valuations` | Admin/Agent | none | none | ❌ |
| 84 | GET | `/api/v1/properties/yield-analysis` | Admin/Agent | none | none | ❌ |
| 85 | GET | `/api/v1/properties/comparables` | Admin/Agent | none | none | ❌ |
| 86 | GET | `/api/v1/properties/market-trends` | Admin/Agent | none | none | ❌ |
| 87 | GET | `/api/v1/properties/insights/market-overview` | Admin/Agent | none | none | ❌ |
| 88 | GET | `/api/v1/properties/insights/area-comparison` | Admin/Agent | none | none | ❌ |
| 89 | GET | `/api/v1/properties/insights/price-trends` | Admin/Agent | none | none | ❌ |
| 90 | GET | `/api/v1/properties/insights/top-areas` | Admin/Agent | none | none | ❌ |
| 91 | GET | `/api/v1/properties/brokers` | Admin/Agent | none | none | ❌ |
| 92 | GET | `/api/v1/properties/transactions/by-project/:name` | Admin/Agent | none | none | ❌ |
| 93 | GET | `/api/v1/properties/transactions/area/:name/summary` | Admin/Agent | none | none | ❌ |
| 94 | GET | `/api/v1/properties/transactions/:id` | Admin/Agent | none | none | ❌ |
| 95 | GET | `/api/v1/properties/transactions/:id/enriched` | Admin/Agent | none | none | ❌ |
| 96 | GET | `/api/v1/properties/rentals/stats` | Admin/Agent | none | none | ❌ |
| 97 | GET | `/api/v1/properties/rentals/areas` | Admin/Agent | none | none | ❌ |
| 98 | GET | `/api/v1/properties/rentals/project/:name` | Admin/Agent | none | none | ❌ |
| 99 | GET | `/api/v1/properties/rentals/building/:name` | Admin/Agent | none | none | ❌ |
| 100 | GET | `/api/v1/properties/lands` | Admin/Agent | none | none | ❌ |
| 101 | GET | `/api/v1/properties/lands/:id` | Admin/Agent | none | none | ❌ |
| 102 | GET | `/api/v1/properties/map/config` | Admin/Agent | none | none | ❌ |
| 103 | GET | `/api/v1/properties/map/bounds` | Admin/Agent | none | none | ❌ |
| 104 | GET | `/api/v1/properties/map/poi-categories` | Admin/Agent | none | none | ❌ |
| 105 | GET | `/api/v1/properties/map/heatmap` | Admin/Agent | none | none | ❌ |
| 106 | GET | `/api/v1/properties/map/area/:name` | Admin/Agent | none | none | ❌ |
| 107 | GET | `/api/v1/properties/areas/:id` | Admin/Agent | none | none | ❌ |
| 108 | GET | `/api/v1/properties/developers/:id` | Admin/Agent | none | none | ❌ |
| 109 | GET | `/api/v1/properties/projects/:id` | Admin/Agent | none | none | ❌ |
| 110 | GET | `/api/v1/properties/units/:id` | Admin/Agent | none | none | ❌ |
| 111 | GET | `/api/v1/properties/valuations/:id` | Admin/Agent | none | none | ❌ |

**Gaps:** 48 property endpoints. Only `search`, `transactions` are called from the FE via direct `fetch()`.

---

## Notifications (`/api/v1/notifications`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 112 | GET | `/api/v1/notifications` | any auth | `api.notifications.list()` | `hooks/useNotifications.ts` | ✅ |
| 113 | PATCH | `/api/v1/notifications/:id/read` | any auth | `api.notifications.markRead()` | `hooks/useNotifications.ts` | ✅ |

---

## Deals (`/api/v1/deals`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 114 | GET | `/api/v1/deals` | any auth | `api.deals.list()` | `(dashboard)/deals/page.tsx` | ✅ |
| 115 | GET | `/api/v1/deals/:id` | any auth | `api.deals.get()` | `deals/[id]/page.tsx` | ✅ |
| 116 | GET | `/api/v1/deals/:id/invoices` | any auth | `api.deals.invoices()` | `deals/[id]/page.tsx` | ✅ |
| 117 | POST | `/api/v1/deals` | Admin/Agent | `api.deals.create()` | none | 🔌 |
| 118 | PATCH | `/api/v1/deals/:id` | Admin/Agent | `api.deals.update()` | `deals/[id]/page.tsx` | ✅ |
| 119 | PATCH | `/api/v1/deals/:id/stage` | Admin/Agent | `api.deals.updateStage()` | none | 🔌 |

---

## Invoices (`/api/v1/invoices`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 120 | GET | `/api/v1/invoices/:id` | any auth | `api.invoices.get()` | none | 🔌 |
| 121 | GET | `/api/v1/invoices/:id/pdf` | any auth | `api.invoices.getPDF()` | `deals/[id]/page.tsx` (URL) | ✅ |
| 122 | POST | `/api/v1/invoices` | Admin/Agent | `api.invoices.create()` | `deals/[id]/page.tsx` | ✅ |
| 123 | POST | `/api/v1/invoices/:id/send` | Admin | `api.invoices.send()` | `deals/[id]/page.tsx` | ✅ |
| 124 | PATCH | `/api/v1/invoices/:id/status` | Admin | `api.invoices.updateStatus()` | `deals/[id]/page.tsx` | ✅ |

---

## Email (`/api/v1/emails`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 125 | POST | `/api/v1/emails/send` | Admin/Agent | `api.email.send()` | none | 🔌 |
| 126 | GET | `/api/v1/emails/history` | Admin/Agent | `api.email.history()` | none | 🔌 |

---

## Rental Properties (`/api/v1/rental-properties`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 127 | GET | `/api/v1/rental-properties` | any auth | `api.rentalProperties.list()` | `(dashboard)/rentals/page.tsx`, leases | ✅ |
| 128 | GET | `/api/v1/rental-properties/:id` | any auth | `api.rentalProperties.get()` | none | 🔌 |
| 129 | POST | `/api/v1/rental-properties` | Admin/Agent | `api.rentalProperties.create()` | `(dashboard)/rentals/page.tsx` | ✅ |
| 130 | PATCH | `/api/v1/rental-properties/:id` | Admin/Agent | `api.rentalProperties.update()` | none | 🔌 |
| 131 | DELETE | `/api/v1/rental-properties/:id` | Admin | `api.rentalProperties.delete()` | none | 🔌 |

---

## Tenants (`/api/v1/tenants`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 132 | GET | `/api/v1/tenants` | any auth | `api.tenants.list()` | `(dashboard)/tenants/page.tsx`, leases | ✅ |
| 133 | GET | `/api/v1/tenants/:id` | any auth | `api.tenants.get()` | none | 🔌 |
| 134 | POST | `/api/v1/tenants` | Admin/Agent | `api.tenants.create()` | `(dashboard)/tenants/page.tsx` | ✅ |
| 135 | PATCH | `/api/v1/tenants/:id` | Admin/Agent | `api.tenants.update()` | none | 🔌 |
| 136 | DELETE | `/api/v1/tenants/:id` | Admin | `api.tenants.delete()` | none | 🔌 |
| 137 | POST | `/api/v1/tenants/:id/verify` | Admin | `api.tenants.verify()` | none | 🔌 |

---

## Lease Templates (`/api/v1/lease-templates`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 138 | GET | `/api/v1/lease-templates` | any auth | `api.leaseTemplates.list()` | none | 🔌 |
| 139 | GET | `/api/v1/lease-templates/:id` | any auth | `api.leaseTemplates.get()` | none | 🔌 |
| 140 | POST | `/api/v1/lease-templates` | Admin | `api.leaseTemplates.create()` | none | 🔌 |
| 141 | PATCH | `/api/v1/lease-templates/:id` | Admin | `api.leaseTemplates.update()` | none | 🔌 |
| 142 | DELETE | `/api/v1/lease-templates/:id` | Admin | `api.leaseTemplates.delete()` | none | 🔌 |

---

## Leases (`/api/v1/leases`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 143 | GET | `/api/v1/leases` | any auth | `api.leases.list()` | `(dashboard)/leases/page.tsx` | ✅ |
| 144 | GET | `/api/v1/leases/:id` | any auth | `api.leases.get()` | `leases/[id]/page.tsx` | ✅ |
| 145 | POST | `/api/v1/leases` | Admin/Agent | `api.leases.create()` | `(dashboard)/leases/page.tsx` | ✅ |
| 146 | PATCH | `/api/v1/leases/:id` | Admin/Agent | `api.leases.update()` | none | 🔌 |
| 147 | DELETE | `/api/v1/leases/:id` | Admin | `api.leases.delete()` | none | 🔌 |

---

## Payments (`/api/v1/payments`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 148 | GET | `/api/v1/payments` | any auth | `api.payments.list()` | `(dashboard)/payments/page.tsx` | ✅ |
| 149 | GET | `/api/v1/payments/:id` | any auth | `api.payments.get()` | none | 🔌 |
| 150 | POST | `/api/v1/payments` | Admin/Agent | `api.payments.create()` | `(dashboard)/payments/page.tsx` | ✅ |
| 151 | PATCH | `/api/v1/payments/:id` | Admin/Agent | `api.payments.update()` | none | 🔌 |
| 152 | DELETE | `/api/v1/payments/:id` | Admin | `api.payments.delete()` | none | 🔌 |

---

## Bank Integrations (`/api/v1/bank-integrations`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 153 | GET | `/api/v1/bank-integrations` | Admin | `api.bankIntegrations.list()` | `(dashboard)/bank-integrations/page.tsx` | ✅ |
| 154 | GET | `/api/v1/bank-integrations/:id` | Admin | `api.bankIntegrations.get()` | none | 🔌 |
| 155 | POST | `/api/v1/bank-integrations` | Admin | `api.bankIntegrations.create()` | none | 🔌 |
| 156 | PATCH | `/api/v1/bank-integrations/:id` | Admin | `api.bankIntegrations.update()` | none | 🔌 |
| 157 | DELETE | `/api/v1/bank-integrations/:id` | Admin | `api.bankIntegrations.delete()` | none | 🔌 |

---

## Bank Statements (`/api/v1/bank-statements`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 158 | GET | `/api/v1/bank-statements` | Admin/Agent | `api.bankStatements.list()` | none | 🔌 |
| 159 | GET | `/api/v1/bank-statements/:id` | Admin/Agent | `api.bankStatements.get()` | none | 🔌 |
| 160 | POST | `/api/v1/bank-statements/upload` | Admin/Agent | `api.bankStatements.upload()` | none | 🔌 |
| 161 | DELETE | `/api/v1/bank-statements/:id` | Admin | `api.bankStatements.delete()` | none | 🔌 |

---

## Payment Confirmations (`/api/v1/payments/:payment_id/confirmation`, `/send-confirmation`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 162 | GET | `/api/v1/payments/:payment_id/confirmation` | Admin/Agent | `api.paymentConfirmations.getByPayment()` | none | 🔌 |
| 163 | POST | `/api/v1/payments/:payment_id/send-confirmation` | Admin/Agent | `api.paymentConfirmations.send()` | none | 🔌 |

---

## Analytics (`/api/v1/analytics`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 164 | GET | `/api/v1/analytics/tenant-overview` | any auth | `api.analytics.getTenantOverview()` | `(dashboard)/analytics/page.tsx` | ✅ |
| 165 | GET | `/api/v1/analytics/properties` | any auth | `api.analytics.listProperties()` | analytics pages | ✅ |
| 166 | GET | `/api/v1/analytics/properties/:propertyID` | any auth | `api.analytics.getProperty()` | `analytics/properties/[id]/page.tsx` | ✅ |
| 167 | GET | `/api/v1/analytics/tenants` | any auth | `api.analytics.listTenants()` | `analytics/tenants/page.tsx` | ✅ |
| 168 | GET | `/api/v1/analytics/tenants/:tenantID` | any auth | `api.analytics.getTenant()` | none | 🔌 |
| 169 | GET | `/api/v1/analytics/financial` | any auth | `api.analytics.getFinancial()` | analytics pages | ✅ |
| 170 | GET | `/api/v1/analytics/maintenance` | any auth | `api.analytics.getMaintenance()` | `(dashboard)/analytics/page.tsx` | ✅ |

---

## Expenses (`/api/v1/expenses`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 171 | GET | `/api/v1/expense-categories` | any auth | `api.expenses.listCategories()` | none | 🔌 |
| 172 | POST | `/api/v1/expense-categories` | Admin | `api.expenses.createCategory()` | none | 🔌 |
| 173 | GET | `/api/v1/expenses` | any auth | `api.expenses.list()` | none | 🔌 |
| 174 | GET | `/api/v1/expenses/:id` | any auth | `api.expenses.get()` | none | 🔌 |
| 175 | POST | `/api/v1/expenses` | Admin/Agent | `api.expenses.create()` | none | 🔌 |
| 176 | PATCH | `/api/v1/expenses/:id` | Admin/Agent | `api.expenses.update()` | none | 🔌 |
| 177 | DELETE | `/api/v1/expenses/:id` | Admin | `api.expenses.delete()` | none | 🔌 |
| 178 | POST | `/api/v1/expenses/:id/approve` | Admin | `api.expenses.approve()` | none | 🔌 |

---

## Inspections (`/api/v1/inspections`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 179 | GET | `/api/v1/inspection-templates` | any auth | `api.inspection.listTemplates()` | none | 🔌 |
| 180 | POST | `/api/v1/inspection-templates` | Admin | `api.inspection.createTemplate()` | none | 🔌 |
| 181 | GET | `/api/v1/inspections` | any auth | `api.inspection.list()` | `(dashboard)/inspections/page.tsx` | ✅ |
| 182 | GET | `/api/v1/inspections/:id` | any auth | `api.inspection.get()` | none | 🔌 |
| 183 | POST | `/api/v1/inspections` | Admin/Agent | `api.inspection.create()` | none | 🔌 |
| 184 | PATCH | `/api/v1/inspections/:id` | Admin/Agent | `api.inspection.update()` | none | 🔌 |
| 185 | POST | `/api/v1/inspections/:id/complete` | Admin/Agent | `api.inspection.complete()` | none | 🔌 |

---

## Maintenance Tasks (`/api/v1/maintenance-tasks`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 186 | GET | `/api/v1/maintenance-tasks` | any auth | `api.maintenance.list()` | `(dashboard)/maintenance/page.tsx` | ✅ |
| 187 | GET | `/api/v1/maintenance-tasks/:id` | any auth | `api.maintenance.get()` | none | 🔌 |
| 188 | POST | `/api/v1/maintenance-tasks` | Admin/Agent | `api.maintenance.create()` | none | 🔌 |
| 189 | PATCH | `/api/v1/maintenance-tasks/:id` | Admin/Agent | `api.maintenance.update()` | none | 🔌 |
| 190 | POST | `/api/v1/maintenance-tasks/:id/complete` | Admin/Agent | `api.maintenance.complete()` | none | 🔌 |
| 191 | POST | `/api/v1/maintenance-tasks/:id/photos` | Admin/Agent | `api.maintenance.addPhoto()` | none | 🔌 |
| 192 | GET | `/api/v1/maintenance-tasks/:id/photos` | any auth | `api.maintenance.getPhotos()` | none | 🔌 |
| 193 | DELETE | `/api/v1/maintenance-tasks/:id` | Admin | `api.maintenance.delete()` | none | 🔌 |

---

## Lease Renewals (`/api/v1/lease-renewals`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 194 | GET | `/api/v1/lease-renewals` | any auth | `api.leaseRenewals.list()` | none | 🔌 |
| 195 | GET | `/api/v1/lease-renewals/:id` | any auth | `api.leaseRenewals.get()` | none | 🔌 |
| 196 | POST | `/api/v1/lease-renewals/:lease_id/initiate` | Admin | `api.leaseRenewals.initiate()` | none | 🔌 |
| 197 | PUT | `/api/v1/lease-renewals/:id/propose` | Admin/Agent | `api.leaseRenewals.propose()` | none | 🔌 |
| 198 | POST | `/api/v1/lease-renewals/:id/send-offer` | Admin/Agent | `api.leaseRenewals.sendOffer()` | none | 🔌 |
| 199 | PUT | `/api/v1/lease-renewals/:id/accept` | Admin/Agent | `api.leaseRenewals.accept()` | none | 🔌 |
| 200 | PUT | `/api/v1/lease-renewals/:id/reject` | Admin/Agent | `api.leaseRenewals.reject()` | none | 🔌 |
| 201 | POST | `/api/v1/lease-renewals/:id/counter-offer` | Admin/Agent | `api.leaseRenewals.counterOffer()` | none | 🔌 |

---

## Renewal Templates (`/api/v1/renewal-templates`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 202 | GET | `/api/v1/renewal-templates` | any auth | `api.leaseRenewals.templates.list()` | none | 🔌 |
| 203 | POST | `/api/v1/renewal-templates` | Admin | `api.leaseRenewals.templates.create()` | none | 🔌 |

---

## Documents (`/api/v1/documents`)

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 204 | GET | `/api/v1/documents/templates` | any auth | `api.documents.templates.list()` | `(dashboard)/documents/page.tsx` | ✅ |
| 205 | GET | `/api/v1/documents/templates/:id` | any auth | `api.documents.templates.get()` | `documents/templates/[id]/page.tsx` | ✅ |
| 206 | POST | `/api/v1/documents/templates` | Admin/Agent | `api.documents.templates.create()` | `(dashboard)/documents/page.tsx` | ✅ |
| 207 | PATCH | `/api/v1/documents/templates/:id` | Admin/Agent | `api.documents.templates.update()` | `documents/templates/[id]/page.tsx` | ✅ |
| 208 | DELETE | `/api/v1/documents/templates/:id` | Admin | `api.documents.templates.delete()` | `(dashboard)/documents/page.tsx` | ✅ |
| 209 | GET | `/api/v1/documents` | any auth | `api.documents.list()` | `components/DocumentAttachmentSection.tsx` | ✅ |
| 210 | GET | `/api/v1/documents/:id` | any auth | `api.documents.get()` | `components/DocumentAttachmentSection.tsx` | ✅ |
| 211 | POST | `/api/v1/documents` | Admin/Agent | `api.documents.create()` | `components/DocumentAttachmentSection.tsx` | ✅ |
| 212 | DELETE | `/api/v1/documents/:id` | Admin/Agent | `api.documents.delete()` | `components/DocumentAttachmentSection.tsx` | ✅ |
| 213 | POST | `/api/v1/documents/:id/request-signature` | Admin/Agent | `api.documents.requestSignature()` | `components/DocumentAttachmentSection.tsx` | ✅ |
| 214 | PATCH | `/api/v1/documents/signatures/:id/mark-signed` | Admin/Agent | `api.documents.markSigned()` | none | 🔌 |

---

## Public / No-Auth Endpoints

| # | Method | Endpoint | Role | FE Wrapper | FE Usage | Status |
|---|--------|----------|------|------------|----------|--------|
| 215 | GET | `/api/public/sign/:id` | public | `api.public.getSignature()` | `sign/[id]/page.tsx` | ✅ |
| 216 | POST | `/api/public/sign/:id` | public | `api.public.sign()` | `sign/[id]/page.tsx` | ✅ |

---

## Webhooks (external — no frontend needed)

| # | Method | Endpoint | Purpose |
|---|--------|----------|---------|
| W1 | GET | `/webhooks/whatsapp` | Meta Cloud API webhook verification |
| W2 | POST | `/webhooks/whatsapp` | Receive WhatsApp messages from Meta |
| W3 | POST | `/webhooks/stripe` | Stripe billing events |
| W4 | POST | `/webhooks/leads` | Public lead intake via API key |

---

## Infrastructure

| # | Method | Endpoint | Purpose |
|---|--------|----------|---------|
| I1 | GET | `/health` | DB + Redis health check |
| I2 | GET | `/docs/*` | Swagger UI (BasicAuth in prod) |
| I3 | WS | `/ws/notifications` | Real-time WebSocket notifications |

---

## Summary

| Status | Count | Description |
|--------|-------|-------------|
| ✅ Connected | 62 | Wrapper exists + used by a page/component |
| 🔌 Wrapper only | 63 | Wrapper exists in `api.ts` but no page calls it yet |
| ❌ Missing | 49 | No FE wrapper (mostly `/api/v1/properties/*` endpoints) |
| 🚫 Webhook/System | 7 | External-only endpoints |

**Total backend endpoints: ~181**
