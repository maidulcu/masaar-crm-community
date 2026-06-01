# Development Phases — PixxiCRM Gap Closure

This document outlines the phased implementation of missing features to bring Masaar CRM to parity with PixxiCRM for UAE real estate brokerages.

---

## Phase 1: Listings Management (Internal)

Build a full listings CRUD — separate from `rental_properties` (which is portfolio/asset management). This is for sales/marketing: agents create listings for properties they're selling/renting on behalf of clients.

**What's needed:**
- New `listings` table (title, description, price, beds/baths/sqft, property type, location, status, agent, featured flag, media)
- Domain model, repo, handler, routes
- Frontend: list page + detail page + create/edit form
- Photo upload support (cover image, gallery)
- Agent assignment per listing
- Status workflow: draft → published → sold/rented → expired

**Deliverables:** 1 migration, ~5 Go files, 2 frontend pages

---

## Phase 2: Portal Sync Engine

Syndicate listings to Property Finder, Bayut, Dubizzle, JamesEdition.

**What's needed:**
- Portal API clients for each portal (PF XML feed, Bayut API, Dubizzle API, JamesEdition API)
- Portal connection config UI (API keys per portal in settings)
- Sync queue: publish, update, delist per listing
- Sync status tracking per portal per listing
- Portal field mapping (amenities → portal-specific categories)
- Media validation (photo size, format requirements per portal)
- Manual "Publish to..." button on listing detail
- Webhook listener for portal sync callbacks

**Deliverables:** Portal API package, settings UI, sync background job, sync status indicators on listings

---

## Phase 3: BOS24 Integration — Deepen

The BOS24 (BuyOrSell24) client exists but needs deeper integration into the CRM workflow beyond the current search/property data pages.

**What's needed:**
- **Auto-populate listings** from BOS24 transaction/project data (agent selects a transaction → pre-fills listing form)
- **Market data on listing detail** — show comparable sales, yield estimates, area stats alongside the listing
- **Listing valuation tool** — use BOS24 valuation endpoint to suggest listing price
- **Area insights widget** on dashboard — top areas, price trends, transaction volumes
- **Property alerts** — notify agents when new listings/transactions match saved search criteria
- **BOS24 data refresh health** in settings (last sync timestamp, data age warnings)
- **Map layer overlay** — show BOS24 transactions on the map (bridge to Phase 5)
- **Improve BOS24 client** with proper typed responses (currently `map[string]interface{}`)
- **Caching strategy** review — Redis TTLs, stale-while-revalidate patterns

**Deliverables:** 3–5 handler enhancements, frontend widgets, typed BOS24 response models, cache tuning

---

## Phase 4: Map View

Build the map frontend — backend endpoints already exist (`/properties/map/*`).

**What's needed:**
- Add Leaflet or Mapbox GL JS dependency
- Map page with property pins (BOS24 transactions + own listings)
- Filters on map: property type, price range, area, bedrooms
- Pin clustering for performance
- Click pin → popup with property summary + link to detail
- Heatmap toggle
- Area boundary overlay

**Deliverables:** 1 frontend page, 1 map component

---

## Phase 5: Offer Management

Track offers made on listings — from initial offer through negotiation to acceptance/rejection. Core brokerage workflow that bridges Listings → Deals.

**What's needed:**
- `offers` table (listing_id, buyer/contact, offer_amount, currency, terms, status, valid_until)
- Offer status workflow: submitted → under_review → countered → accepted → rejected → expired
- Counter-offer chain (parent_offer_id for threading)
- Offer creation from listing detail or lead profile
- Notification to agent + owner on new offer
- Auto-convert accepted offer to a Deal
- Offer summary on listing detail page
- Required documents per offer (cheque copy, ID, etc.)

**Deliverables:** 1 migration, handler + repo, UI on listing detail + offer list page

---

## Phase 6: Approval Workflows

Beyond expense approvals — listing publishing approval, deal sign-offs, offer acceptance limits, permission-based controls per agent tier.

**What's needed:**
- **Listing approval** — draft → submitted_for_approval → approved → published
- **Deal sign-off** — manager must approve deals above a threshold value
- **Offer acceptance limits** — agents auto-accept below X AED, require approval above
- **Configurable approval chains** — per company, per role
- **Approval notifications** — approver gets notified, approver action notifies requester
- **Audit trail** for all approvals (extends existing audit log)
- **Permission presets** — by role (junior agent, senior agent, manager, director)

**Deliverables:** 1 migration, approval engine service, handler updates, settings UI

---

## Phase 7: Lead Rotation Engine

Automated round-robin lead distribution.

**What's needed:**
- `lead_assignment_mode` setting already in seed data — consume it
- Round-robin algorithm: track last assigned agent per team, cycle to next
- Smart rules: capacity-based (cap per agent), skills-based (property type match)
- Admin config UI for rotation rules
- Override: manual assign still allowed
- Notification to agent on auto-assign

**Deliverables:** 1 service file, 1 settings page section, updates to `lead.go` handler

---

## Phase 8: Commission Engine (Complete)

Schema exists (migration 0028) — handlers and UI are missing.

**What's needed:**
- Commission structure CRUD (fixed/percentage/tiered)
- Auto-calculation on deal close / lease sign
- Agent commission dashboard: earned, pending, paid, disputed
- Payout management: mark paid, generate payment reference
- Reports: commission by agent, by period, by deal type
- Admin approval workflow for large commissions

**Deliverables:** 2–3 handler files, repo methods, 2 frontend pages, 1 migration (minor)

---

## Phase 9: Agent Performance & Gamification

Leaderboards, targets, rankings — motivate agents and give managers visibility.

**What's needed:**
- **Agent performance dashboard** — listings added, deals closed, revenue generated, leads converted
- **Monthly/quarterly targets** — set per agent, track progress %
- **Leaderboard** — rank agents by KPIs (listings, deals, revenue, response time)
- **Badges & achievements** — "Top Lister", "Deal Closer", "Fast Responder"
- **Performance export** — PDF/CSV report for management reviews
- **KPI trends** — compare this month vs last month, vs same month last year

**Deliverables:** 1–2 handler files, 2 frontend pages, performance service

---

## Phase 10: Calendar & Viewings

Track viewings, tasks, and meetings with reminders.

**What's needed:**
- `viewings` table (listing, client, agent, datetime, status, notes)
- Calendar view (month/week/day) — use a library like FullCalendar
- Viewing creation from listing detail or lead profile
- Reminders via notification + email
- Google Calendar sync (optional, Phase 10b)
- Conflict detection (agent double-booking)
- Check-in/check-out for property access tracking

**Deliverables:** 1 migration, handler + repo, 1 frontend page, 1 calendar component

---

## Phase 11: Custom Workflow Builder

Drag-and-drop pipeline stage editor.

**What's needed:**
- Pipeline stages no longer hardcoded — stored per company
- Drag-and-drop UI to reorder stages, add/remove
- Stage rules: required fields per stage, auto-assign on entry
- Deal pipeline also user-configurable
- Backwards compatibility: seed default stages

**Deliverables:** 1 migration (stages table), handlers, 1 frontend page, drag-drop component

---

## Phase 12: Property Marketing Tools

Generate PDF brochures, shareable listing landing pages, and marketing collateral directly from the CRM.

**What's needed:**
- **PDF brochure generator** — listing detail → branded PDF with photos, specs, agent info
- **Shareable listing page** — public URL for each listing (no login required)
- **QR code generation** — per listing for flyers/signboards
- **Social media card** — auto-generated OG image for WhatsApp/LinkedIn sharing
- **Email campaign** — send listing to contact list with tracking
- **Branding config** — company logo, colors, disclaimer text on marketing outputs

**Deliverables:** PDF template engine, public listing route, QR package, branding settings

---

## Phase 13: Customer / Owner Portal

Let property owners log in, view their listing status, lease info, payment history, and documents.

**What's needed:**
- **Owner user role** — limited to own properties only
- **Owner dashboard** — my listings (status, views, offers), my leases, my invoices
- **Document access** — owner can view/sign lease documents
- **Payment history** — rent payments received, pending, overdue
- **Communication log** — messages from agent about their property
- **Owner onboarding** — invite owner via email, set portal password
- **Separate login** for owners (not the same as agent/admins)

**Deliverables:** Owner role + JWT, restricted data scope, 2–3 frontend pages, invite flow

---

## Phase 14: Data Import / Export

Bulk import/export contacts, listings, leads, and properties. Schema ready (migration 0031 `bulk_operations`) but no handler/UI.

**What's needed:**
- **CSV import** — contacts, listings, leads with column mapping UI
- **CSV export** — filtered lists → downloadable CSV
- **Import validation** — preview rows, show errors before committing
- **Import history** — track imports, rollback on failure
- **Export templates** — pre-configured export formats (e.g., for portal uploads)
- **Large file handling** — chunked processing via background job (schema exists)

**Deliverables:** 1 handler file, 1 frontend page, background job consumer, validation service

---

## Phase 15: ManyChat / Chatbot Integration

AI chatbot for lead capture on website + WhatsApp.

**What's needed:**
- ManyChat webhook receiver (or generic chatbot webhook)
- Auto-reply rules based on keywords
- Lead creation from chatbot conversation
- WhatsApp auto-reply during business hours (setting already seeded)
- Conversation flow builder (simple if/then or AI-powered)

**Deliverables:** 1 webhook handler, auto-reply service, settings UI

---

## Phase 16: Native Zapier Integration

OAuth 2.0-based Zapier app for one-click connections.

**What's needed:**
- OAuth 2.0 provider (auth code flow)
- Zapier app definition (JSON)
- Subscribe/unsubscribe webhook management for Zapier triggers
- Trigger endpoints: new lead, new deal, new listing
- Action endpoints: create lead, create contact, create listing

**Deliverables:** OAuth handler, Zapier app definition, trigger + action handlers

---

## Execution Order

```
Phase 1  ─────────────────►  Listings (1 week)
Phase 2  ─────────────────►  Portal Sync (2 weeks)
Phase 3  ─────────────────►  BOS24 Deepen (1 week)
Phase 4  ─────────────────►  Map View (1 week)
Phase 5  ─────────────────►  Offer Management (1 week)
Phase 6  ─────────────────►  Approval Workflows (1 week)
Phase 7  ─────────────────►  Lead Rotation (3 days)
Phase 8  ─────────────────►  Commission Engine (1 week)
Phase 9  ─────────────────►  Agent Performance (1 week)
Phase 10 ─────────────────►  Calendar & Viewings (1 week)
Phase 11 ─────────────────►  Workflow Builder (1 week)
Phase 12 ─────────────────►  Marketing Tools (1 week)
Phase 13 ─────────────────►  Owner Portal (2 weeks)
Phase 14 ─────────────────►  Import / Export (1 week)
Phase 15 ─────────────────►  Chatbot (1 week)
Phase 16 ─────────────────►  Zapier (1 week)
```

---

## Dependencies

```
Phase 1 (Listings) → Phase 2 (Portal Sync), Phase 5 (Offers), Phase 10 (Viewings), Phase 12 (Marketing)
Phase 5 (Offers)   → Deal creation (Deals already done)
Phase 6 (Approval) → Phase 1 (listing approval), Deals (deal sign-off)
Phase 8 (Commission) → Deals + Leases (already done)
Phase 3 (BOS24)    → Standalone — can progress in parallel with Phases 1-2
Phase 4 (Map View) → Backend already done, standalone frontend work
Phase 7 (Lead Rotation) → Standalone
Phase 9 (Performance) → Phase 8 (commission data for revenue)
Phase 11 (Workflow) → Standalone
Phase 12 (Marketing) → Phase 1 (listing data)
Phase 13 (Portal)  → Phase 1 (listing data), Phase 5 (offers), Phase 10 (viewings)
Phase 14 (Import)  → Standalone
Phase 15 (Chatbot) → Standalone
Phase 16 (Zapier)  → API keys (already done)
```

---

## Parallel Tracks

Some phases can run in parallel since they touch different parts of the codebase:

```
Track A (Sales):    Phase 1 → Phase 2 → Phase 5 → Phase 6
Track B (Property): Phase 3 → Phase 4 → Phase 12
Track C (People):   Phase 7 → Phase 8 → Phase 9 → Phase 10
Track D (Platform): Phase 11 → Phase 13 → Phase 14 → Phase 15 → Phase 16
```
