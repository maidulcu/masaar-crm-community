# Remaining Tasks

## Priority Legend

| Label | Meaning |
|-------|---------|
| **P0** | Core CRM feature, users will notice it's missing |
| **P1** | Important enhancement, improves existing workflow |
| **P2** | Nice to have, fills a gap but not critical |
| **P3** | Polish / cleanup |

---

## 1. AI Features with No UI

### P0 — AI Lead Scoring (`POST /ai/score-lead/:id`)
- **Backend**: Endpoint exists, `api.ai.scoreLead()` wrapper exists
- **Frontend**: **Nowhere called**. No button or display for lead scores.
- **What to build**: A "Score Lead" button on the pipeline Kanban card / lead detail page. Display the score (0-100) as a badge or progress bar.
- **Files involved**: `pipeline/page.tsx`, `contacts/[id]/page.tsx`, `lib/api.ts` (wrapper already exists)

### P1 — AI Describe Listing (`POST /ai/describe-listing`)
- **Backend**: Endpoint exists, `api.ai.describeListing()` wrapper exists
- **Frontend**: **Nowhere called**. No UI to generate property descriptions.
- **What to build**: A "Generate Description" button on the rental property detail page (`rentals/[id]/page.tsx`) or properties page. Shows AI-generated description in a modal/textarea.
- **Files involved**: `rentals/[id]/page.tsx`, `properties/page.tsx`

### P1 — Message Analysis (`POST /messages/analyze`)
- **Backend**: Endpoint exists, `api.messages.analyze()` wrapper exists
- **Frontend**: **Nowhere called**. No UI to analyze a message.
- **What to build**: An "Analyze" button on individual messages in the inbox thread detail page. Shows intent, sentiment, entities in a modal.
- **Files involved**: `inbox/[id]/page.tsx`

---

## 2. Missing List / Detail Pages

### P1 — Invoices Standalone Page
- **Backend**: Full CRUD: `GET /invoices/:id`, `POST /invoices`, `POST /invoices/:id/send`, `PATCH /invoices/:id/status`, `GET /invoices/:id/pdf`
- **Frontend**: Invoice management is **only** embedded inside `deals/[id]/page.tsx`. No `/invoices` list page or standalone `/invoices/[id]` detail page.
- **What to build**:
  - `invoices/page.tsx` — list with status filter + create button
  - `invoices/[id]/page.tsx` — detail with send/update-status/pdf-download actions
- **Sidebar**: Add "Invoices" link under CRM or Account section.

### P1 — Email History Page
- **Backend**: `GET /emails/history` endpoint exists, `api.email.history()` wrapper exists
- **Frontend**: **Nowhere called**. No email history view anywhere.
- **What to build**: An email history page at `/emails` showing sent emails with recipient, subject, status, timestamp. Filterable by entity.
- **Sidebar**: Add "Emails" link or embed in contact detail page.

### P1 — Message Templates Management Page
- **Backend**: Full CRUD (`GET/POST/PATCH/DELETE /message-templates`)
- **Frontend**: Read-only usage via `TemplatePicker` component (used in inbox for template messages). **No create/edit/delete UI**.
- **What to build**: An admin management page at `/settings/message-templates` with a form to create/edit WhatsApp message templates (name, body with variable placeholders) and delete button.
- **Sidebar**: Add under Account/Admin section.

### P2 — Renewal Templates Management Page
- **Backend**: `GET /renewal-templates` and `POST /renewal-templates`
- **Frontend**: API wrapper exists (`api.leaseRenewals.templates.*`) but **no UI to create templates**.
- **What to build**: A section on the renewals page or a standalone admin page to create renewal templates (name, email subject/body, WhatsApp message, language). Currently templates can only be listed/selected in the send-offer flow.
- **Sidebar**: Add under Properties section or embed in renewals page.

### P2 — Inspection Templates Management Page
- **Backend**: `GET /inspection-templates` and `POST /inspection-templates`
- **Frontend**: Templates can be listed and selected during inspection creation (dropdown). **No UI to create templates**.
- **What to build**: A form on the inspections page (or `/inspections/templates` route) to create inspection templates with a name and checklist items.
- **Files involved**: `inspections/page.tsx` or new page

---

## 3. Sidebar Gaps

Items that have pages but no sidebar entry:

| Route | Sidebar Section | Should Add? |
|-------|----------------|-------------|
| `/analytics/properties` | — | Already under analytics sub-nav |
| `/analytics/tenants` | — | Already under analytics sub-nav |
| `/analytics/financial` | — | Already under analytics sub-nav |
| `/bank-integrations` | — | No sidebar link (admin feature, but page exists) |
| `/audit-log` | Account | ✅ Already added |

Items with pages that DO have sidebar entries but might need reorganization:
- All Account section items (Settings, Company, API Keys, Webhooks, Notifications, Audit Log) are present ✓
- All CRM section items (Dashboard, Pipeline, Inbox, Contacts, Deals) are present ✓
- All Properties section items (Search, Rentals, Tenants, Leases, Lease Templates, Documents, Renewals, Expenses, Payments, Bank Statements, Inspections, Maintenance) are present ✓

---

## 4. Missing API Wrappers

These backend routes have **no frontend API wrapper** in `web/lib/api.ts`:

| Backend Route | Missing Wrapper | Impact |
|---------------|-----------------|--------|
| `GET /properties/areas` | `api.bos24.areas` | Low (BOS24 feature, advanced) |
| `GET /properties/buildings` | `api.bos24.buildings` | Low |
| `GET /properties/projects` | `api.bos24.projects` | Low |
| `GET /properties/developers` | `api.bos24.developers` | Low |
| `GET /properties/units` | `api.bos24.units` | Low |
| `GET /properties/valuations` | `api.bos24.valuations` | Low |
| `GET /properties/market-trends` | `api.bos24.marketTrends` | Low |
| `GET /properties/comparables` | `api.bos24.comparables` | Low |
| `GET /properties/brokers` | `api.bos24.brokers` | Low |
| `GET /properties/map/*` | `api.bos24.map*` | Low |
| `GET /users/me` | Wrapper exists (`api.users.me()`), check usage | Used in auth store |
| `GET /analytics/maintenance` | `api.analytics.getMaintenance()` wrapper exists but **unused** | Add to analytics pages |
| `POST /properties/ai/describe` | Missing | Low |
| `POST /properties/projects/search` | Missing | Low |

---

## 5. Known Compilation Errors (Pre-existing)

These are compile errors that exist in the codebase (NOT caused by recent changes):

| File | Error | Severity |
|------|-------|----------|
| `internal/api/handler/ai.go:188` | `undefined: log` | **Blocks backend build** |
| `internal/api/handler/document.go:42,92,177` | `uuid.Parse` assigns to 1 var (returns 2) | Blocks build |
| `internal/api/handler/expense.go:33,51,87,142` | `uuid.Parse` assigns to 1 var | Blocks build |
| `internal/api/handler/inspection.go:34,52` | `uuid.Parse` assigns to 1 var | Blocks build |
| `web/app/(dashboard)/contacts/[id]/page.tsx:82,250` | `"ar" \| "en"` not assignable to `"en"` | TypeScript error |
| `web/app/(dashboard)/pipeline/page.tsx:510,523` | `Cannot find name 'isAgent'` | TypeScript error |
| `web/app/(dashboard)/settings/billing/page.tsx:126,140` | `'r' is of type 'unknown'` | TypeScript error |

These should be fixed before any production deployment.

---

## 6. Polish & UX Improvements

### P3 — Date Formatting Consistency
- Many pages use `toLocaleDateString` inline; extract to a shared utility function
- Some pages show raw ISO timestamps

### P3 — Loading States
- Some pages lack skeleton loaders
- Add consistent loading states across all list pages

### P3 — Empty States
- Some list pages show nothing when empty (just blank)
- Add "No data" illustrations / CTAs

### P3 — Confirmation Dialogs
- Delete actions on some pages lack confirmation modals
- Add shared `ConfirmDialog` component

### P3 — Pagination Component
- Multiple pages implement pagination inline (load-more, prev/next)
- Extract a shared `Pagination` component

### P3 — Filter Components
- Filter dropdowns, search inputs duplicated across pages
- Extract shared filter bar component

---

## 7. Potential Future Features (Not Yet Required)

These are NOT gaps — they don't have backend endpoints yet:

- Public website / landing page
- Mobile app
- Email templates (for sending) — separate from message templates
- Bulk import (contacts, leads, properties)
- Calendar / task management
- Reporting dashboard (custom reports)
- Two-factor authentication
- SSO / OAuth login
