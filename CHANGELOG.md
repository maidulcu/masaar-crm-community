# Changelog

All notable changes to Masaar CRM will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [v0.4.0] - Unreleased

### Account security

- **Two-factor reset by an admin** (`POST /users/{id}/2fa/reset`) no longer works on the admin's own account: turning off your own two-factor still needs your password and a code (`/users/me/2fa/disable`). Resetting another user's two-factor now also ends that user's sessions. The Users page hides the button on your own row.
- **Audit log:** creating a user directly (`POST /users`) and creating or revoking an API key are now recorded. Key entries hold the name, prefix and scopes, never the key.
- **Revoking an API key** that doesn't exist (or belongs to another company) answers 404 instead of 204.
- **Password reset** mail is no longer sent to deactivated accounts (the answer is unchanged).
- `POST /users` limits `name` to 100 characters and `wa_number` to 20.
- Removed two unused user-repository methods that created users outside any company scope.

### Fixed: new companies had an empty pipeline

- Companies created after the custom pipeline stages were added (sign-up, Partner API) had no stages, so the Pipeline page drew no columns even though their leads were stored. Listing stages now gives a company the default six (`new` … `lost`) the first time it has none, for leads and deals. No migration; existing companies are unaffected.

### Sign-up can be hidden in the web app

- **`NEXT_PUBLIC_SIGNUP_ENABLED=false`** (build time) hides the sign-up links on the login page and makes `/signup` say registration is closed, for instances whose accounts are created elsewhere (pair it with `ALLOW_REGISTRATION=false`, the API default). Defaults to `true`; nothing changes otherwise.
- Optional **`NEXT_PUBLIC_ACCOUNT_PROVIDER_NAME`** / **`NEXT_PUBLIC_ACCOUNT_PROVIDER_URL`**: with sign-up off, the login page and `/signup` tell visitors that accounts come from that service and link to it.

### Outbound webhooks signed per Standard Webhooks (integration standards, step 3)

- **Every delivery carries `webhook-id`, `webhook-timestamp` and `webhook-signature`** ([Standard Webhooks](https://www.standardwebhooks.com)): `v1,` + base64 HMAC-SHA256 of `id.timestamp.body`. Receivers can verify with any Standard Webhooks library; signatures match the spec's reference vector.
- **`webhook-id` is the same on every retry** (and on every subscriber of that event), so receivers can drop duplicates. `X-Masaar-Delivery` now carries the same id instead of a new UUID per attempt.
- **New secrets look like `whsec_…`** (32 random bytes). Secrets created earlier keep working: the libraries read them the same way, and `X-Masaar-Signature` / `X-Masaar-Event` are still sent unchanged.
- The payload gains `"type"` (Standard Webhooks name) next to the existing `"event"`.
- `/openapi.json` documents the events in its OpenAPI 3.1 `webhooks` section (headers, payload and `data` schema per event).
- Docs corrected: `payment.received`, `lease.signed` and `contact.created` were listed as webhook events but are never sent; they are now marked as reserved.

### OpenAPI 3.1 at `/openapi.json` (integration standards, step 2)

- **`GET /openapi.json`** serves an OpenAPI 3.1 description of the API, without authentication, for Postman, Zapier, Make and code generators. It documents 349 operations, each with an `operationId`; API-key routes list their scope (`ApiKeyAuth`, `x-api-key-scope`), `Idempotency-Key` and the RFC 9457 `Problem` body. It passes the Redocly linter.
- **Generated, not hand-written:** `scripts/openapi.sh` runs swag (pinned v1.16.6) on the handler annotations, then `cmd/openapi` converts the Swagger 2.0 output to 3.1, checked against the live router: annotations written with or without the `/api/v1` prefix resolve to the real path, the session scheme is named consistently, public routes are marked public.
- **The Swagger docs were regenerated** for the first time since the first release (27 operations → 349); `/docs` now shows the current API. A few annotations that stopped generation were fixed.
- **CI** fails when `docs/` is stale; `TestOpenAPIMatchesRouter` fails on an annotation for a route that does not exist, or when the number of undocumented routes (56 today: commissions, offers, viewings, performance, import/export, billing) grows.
- CORS allows the `Idempotency-Key` header.

### Market data works: client for the Waqov DLD data API (build plan PR 8)

- **`internal/bos24` is now a real client** for the Waqov real-estate data API ([OpenAPI spec](https://dldapi.waqov.com/openapi.json)), replacing the stub. The Properties and Map pages and the 47 `/api/v1/properties/*` endpoints work once `BOS24_API_TOKEN` is set; without it they still answer `503`.
- Filters are translated to the API's names (`trans_type` → `trans_group`, `limit` → `per_page` on paged lists); path segments are escaped; responses are capped at 10 MB.
- GET responses are cached in Redis (10 min to 24 h by endpoint). Upstream "not found" now answers `404` and rate limiting `429` instead of `503`.
- `POST /api/v1/properties/ai/describe` forwards only the fields the API accepts (`property_id`, `property_data`, `tone`, `language`) and returns `{"description": "..."}` as documented.

### API keys reach the REST API, with scopes (integration standards, step 1)

- **API keys can call `/api/v1`**, limited by scope: `leads:read`, `leads:write`, `contacts:read`, `contacts:write`, `listings:read`, `webhooks:manage` (plus `lead:create` for `POST /webhooks/leads`). Only the routes listed in `internal/api/apikey_routes.go` accept a key; every other `/api/v1` route answers `403`. Data stays limited to the key's company, writes are audited under the key (`audit_logs.api_key_id`), and each key keeps its 300 requests/minute limit. The demo company's keys are read-only. No migration.
- **New keys must use known scopes.** `POST /api/v1/settings/api-keys` refuses unknown scopes and wildcards (`422`); `GET /api/v1/settings/api-keys/scopes` lists them. Existing keys keep working, including old `*` and `resource:*` values.
- **Standard error bodies** for API-key requests: RFC 9457 problem details (`application/problem+json`: `type`, `title`, `status`, `detail`). The `error` field stays, so existing clients keep working. Missing scope → `WWW-Authenticate: Bearer error="insufficient_scope"`, bad key → `error="invalid_token"` (RFC 6750). Applies to `POST /webhooks/leads` too.
- **`Idempotency-Key`** on API-key `POST`/`PATCH` (including `POST /webhooks/leads`): a retry with the same key and request replays the first response (`Idempotent-Replayed: true`); a different request with the same key gets `422`, one still running `409`. Stored 24 hours per key in Redis; a failed or crashed request frees its key; without Redis requests run normally.
- Webhook subscription create/delete and lead note edits are now written to the audit log.
- **Settings → API Keys fixed:** the list was always empty, a new key was never shown, and every key was labelled "Revoked". Scopes are now picked with checkboxes.

### UAE e-invoicing and Saudi groundwork (#88) — migration `0078`

**UAE e-invoicing (PINT AE)** — off by default; see [docs/einvoicing.md](docs/einvoicing.md)
- Tax invoices and credit notes can be exported as **PINT AE** XML (UBL 2.1, the UAE Peppol specification), including mixed standard/exempt/out-of-scope lines and credit note reason codes. Missing data (TIN, legal registration, buyer address…) is listed instead of producing an invalid file.
- The generated XML is checked in CI against the UBL 2.1 schema and the official PINT AE Schematron (`scripts/pint-ae/validate.sh`).
- Submission through a provider **connector** interface, with a status job that records accepted and rejected e-invoices and notifies the issuer of a rejection. Only a **mock** provider ships; a real accredited service provider needs its own connector.
- Settings → E-invoicing (provider, TIN, legal registration, exemption code); an E-invoice panel on each invoice for the buyer details, XML download and submission.
- AED invoices only for now.

**Saudi Arabia groundwork** — see [docs/adr/0001-ksa-groundwork.md](docs/adr/0001-ksa-groundwork.md)
- Country **SA**: 15 % VAT, SAR, Asia/Riyadh, Friday–Saturday weekend, the 13 regions.
- Listings in Saudi regions require a **REGA** advertising licence, enforced like Trakheesi/Madhmoun.
- Optional **Hijri (Umm al-Qura)** dates next to Gregorian dates (Settings → Company).
- Money defaults (new leads, deals, offers, expenses, listings, imports) follow the company currency instead of AED.
- ZATCA e-invoicing, Ejar and Nafath are not part of the community edition.

#### Upgrade notes
- Nothing changes until an admin enables e-invoicing; existing invoices are unaffected.

### KYC/AML, WhatsApp broadcasts and two-factor authentication (#87) — migrations `0075`–`0077`

**Two-factor authentication**
- Optional per user, with an authenticator app (TOTP, RFC 6238): turn on from Settings with a QR code; 8 single-use backup codes (stored hashed); turning it off needs the password and a code.
- Every sign-in path (password, magic link, SMS code) answers with a short-lived pending token instead of a session when a second factor is needed; `POST /auth/verify-totp` exchanges it with a code or a backup code. A code cannot be used twice. Five wrong codes lock the account for 15 minutes (the same lock as wrong passwords).
- Admins can require two-factor for the whole company (Settings → Security); users without it set it up at their next sign-in. Admins can reset a user's two-factor (lost phone).
- Secrets are encrypted at rest with AES-GCM; set `TWO_FACTOR_KEY` (otherwise derived from `JWT_SECRET`).

**KYC and AML (record keeping and workflow; no automatic compliance decisions)**
- Identity documents per contact (Emirates ID, passport, residence visa, trade licence; number, issuing country, dates, a private copy), risk rating with reason, PEP flag, and screening result entered by hand. A screening-provider interface (`kyc.Screener`) is available; none is bundled.
- Beneficial owners of corporate clients (organisations), with their KYC status.
- Alerts before a client's identity document expires.
- **A deal cannot be marked won** while the buyer has no valid Emirates ID or passport on file, unless an admin overrides it with a reason (audit-logged).
- Deals record their payment method (cash, cheque, transfer, virtual assets, mixed) and cash part. Cash at or above the configured threshold (AED 55,000 by default) or payment in virtual assets sends the deal to the **AML review queue** (Compliance page), where the decision (no report / reported, with the goAML reference) is recorded. A CSV export gathers what a goAML report needs; it is not a goAML XML submission.
- A **compliance** role: reads like a viewer, plus the KYC and AML screens. Viewing identity documents and the AML queue is audit-logged.

**WhatsApp**
- **Broadcasts** of an approved template to a segment (lead tag, stage, source, listing interest), only to contacts who granted WhatsApp consent; the others are recorded as skipped. Sent by a background job at a configurable rate per minute, with per-recipient delivery status from Meta's receipts. Consent is checked again just before each message.
- Replying **STOP / إيقاف** (or UNSUBSCRIBE, إلغاء الاشتراك) records a WhatsApp opt-out and confirms it; **START / اشتراك** opts back in. Only whole-message replies count.
- Inbox: the **24-hour customer-service window** is shown on each conversation, and free-form messages outside it are refused with a clear message (an approved template is needed); conversations can be **assigned** to an agent.
- **Business-hours auto-reply**: weekly hours in the company timezone plus special periods (Ramadan hours, holidays); outside them a customer's message gets the auto-reply, at most once every 12 hours per conversation. Settings → WhatsApp Inbox.

#### Upgrade notes
- Set `TWO_FACTOR_KEY` to a long random value before anyone turns on two-factor, and keep it stable.
- Existing deals have no payment method recorded and are not flagged until one is set.

### Property management (#86) — migration `0074`

**Post-dated cheque register**
- Cheques per lease: number, bank, drawer, amount, date, and an optional scan (JPEG, PNG or PDF, kept in the bank statement store). Status: in hand (`received`) → `deposited` → `cleared`, or `bounced`; also `on_hold`, `returned` and `replaced`. Every change is audit-logged.
- **Cheque schedule:** `POST /leases/:id/cheques/schedule` (or `cheque_schedule` in the lease-creation body) splits the rent for the lease term over N cheques at equal month intervals, numbered upwards from the first cheque number, and creates a pending payment for each. A 12-cheque lease gets 12 cheques and 12 payments; rounding goes to the last cheque.
- A cleared cheque marks its payment received (with the clearing date); a bounced cheque marks it failed, notifies the lease's agent and the admins, and can add a returned-cheque charge as a new payment. A replacement cheque pays the same payment.
- **Cheque register** (`/cheques`): cheques to deposit today, in the next 30 days, awaiting clearance, or bounced, with totals and bulk "mark deposited". A daily digest of the cheques to deposit goes to each lease's agent and the admins (`cheque_deposits` job).

**Lease alerts and notices**
- Leases have an **agent** (defaults to whoever created the lease) and an **Ejari expiry**.
- Alerts to the agent and the admins, once per threshold: before a lease ends (by default 120, 90, 60 and 30 days, configurable), two weeks before the last day to serve a rent-change notice when none has been recorded (90 days before the end by default, configurable), and 30 and 7 days before (and once after) the Ejari registration expires (`lease_alerts` job). Clicking the notification opens the lease.
- **Notices to the tenant** (rent change, renewal terms, non-renewal, other): drafted in English and Arabic from the lease, edited, saved, printed side by side, and recorded as served (when and how: email, WhatsApp, by hand, registered mail, notary). The drafts are a starting point, not legal advice.
- "Start renewal" on the lease page opens the existing renewal workflow. Its renewal date is now the day after the lease ends (it was "today + 90 days").

**Rent reminders**
- When switched on (Settings → Property Management), tenants are reminded by email and WhatsApp a few days before rent is due (3 by default) and once when it is late, between 9:00 and 20:00 local time (`rent_reminders` job). WhatsApp uses an approved template with four parameters (tenant name, amount, due date, property).
- Not sent for payments covered by a cheque in hand, to tenants switched off on the lease page, or on a channel the tenant opted out of as a contact (matched by phone or email). Skipped and failed reminders are recorded and shown on the lease page.
- This replaces the no-op `PaymentReminderService` stub; the `payment_reminders` table it expected is created by this migration.

**Other**
- The lease page loads its own payments (`GET /leases/:id/payments`); it used to filter the first 100 payments of the company.
- Notifications open the record they are about (lease, task, listing…).

#### Upgrade notes
- Existing leases get their creator as agent. Rent reminders are off until an admin switches them on.
- Notice periods and alert days are defaults for Dubai: check the rules that apply to your emirate.

### Brokerage workflow (#85) — migrations `0070`–`0073`

**Tasks and follow-ups**
- Tasks (call, meeting, viewing, to-do, follow-up) with a due date, priority and assignee, linked to a lead, contact, deal, listing, viewing or offer. A **Tasks** page (mine / all / overdue), task panels on contact, deal and listing pages, and an overdue widget on the dashboard.
- The assignee is reminded (in-app, live, and by email when mail is configured) about 15 minutes before a task is due, once; moving the due date or the assignee re-arms the reminder (`task_reminders` job).

**Contacts**
- Contact profile: secondary phones, nationality (ISO country), source, preferred contact time, address, job title, notes, and an **organisation** (B2B companies with trade licence and TRN; `/organisations`).
- **Custom fields** for contacts, leads, deals and listings (text, number, date, select, yes/no), managed under Settings → Custom Fields, validated on save, and included in CSV import/export as `cf_<name>` columns. The tables existed since `0030` but were never wired up.
- **Duplicate detection and merge** (`GET /contacts/duplicates`; `POST /contacts/merge`, admin only): contacts sharing an email, a name, or a phone are grouped; merging moves leads, offers, viewings, conversations, consent history, tasks and custom values to the contact you keep.

**RERA forms (Dubai)**
- Pre-filled **Form A** (seller/landlord – broker), **Form B** (buyer/tenant – broker), **Form I** (broker – broker) and **Form F** (MOU between seller and buyer) from the company, listing, agent, contact and offer. Each form is stored as a document, printable in English and Arabic, and can be sent for e-signature through the existing signing link. Form F is offered on accepted offers.
- These are agreement records with the forms' key fields, not the official DLD templates, and nothing is submitted to the DLD. Check the current official forms before relying on them.

**Property portals**
- **Lead webhook** per portal (Property Finder, Bayut, Dubizzle, or any other): `POST /webhooks/portals/:portal?token=…`. The contact is matched or created by phone, the lead gets the portal as its source, a listing reference is noted on the lead, lead rotation assigns it, and `lead.created` is delivered to webhook subscribers. A lead delivered twice is created once.
- **Listing feed** per portal: `GET /api/public/feeds/:portal/listings.xml?token=…` lists published listings with photos, agent (with BRN) and company (with ORN). Listings whose advertising permit is missing or expired are left out.
- Tokens are created, rotated, paused and revoked under **Settings → Property Portals** (admin); they are stored hashed and shown once.
- The portals' own lead and feed formats are not public. The webhook recognises common field names, and the feed uses a documented, generic XML format (`masaar-listings/1`). Mapping to a portal's exact schema is left for contributors with access to its documentation.
- **Lead-source report** (`GET /reports/lead-sources`, Reports → Lead Sources): leads, wins, conversion and won deal value by source for a date range.

**Calculators**
- Buyer's purchase costs (DLD transfer fee, admin and trustee fees, commission, mortgage registration and bank fees), mortgage repayment, rental yield (gross and net) and commission with VAT. The rates are editable and remembered in the browser. Results can be printed or shared on WhatsApp. Linked from listings and offers. The default rates are indicative and should be checked against current DLD fees and the bank's offer.

#### Upgrade notes
- Set `PUBLIC_API_URL` to the API's public address (for example `https://api.crm.example.ae`) so the portal URLs shown to admins are correct behind a proxy. When it is empty, the address of the incoming request is used.
- Migration `0071` deletes custom field values whose definition no longer exists, and keeps one value per field and record.

### UAE compliance core (#84) — migrations `0067`–`0069`

**Tax invoices**
- Invoices have **line items** with a VAT category each (standard, zero-rated, exempt, out of scope), VAT computed and rounded per line under the company's country rules, and stored totals (no longer generated columns).
- The supplier (name, address, TRN) and customer (name, address, TRN for B2B) are **captured when the invoice is issued** and never change afterwards. The customer defaults to the deal's contact.
- **Credit notes** (`CN-YYYY-NNNN`) against an invoice, in full or in part. An invoice can never be credited beyond its total, also under concurrent requests. Invoices show what has been credited and what is outstanding.
- **Void** with a reason (not for paid documents, nor for invoices with credit notes). A void document never changes again.
- **Gap-free numbering** per company, document kind and year, taken inside the issuing transaction (the previous `MAX()+1` could hand two requests the same number).
- A company **without a TRN** can only issue invoices without VAT (out of scope), which are titled "Invoice" rather than "Tax Invoice".
- A **printable bilingual (English | Arabic) invoice** page (`/invoices/<id>/print`, "Print / Save as PDF"), with a VAT summary per category. Server-side PDF generation remains a Pro feature.
- The legacy `POST /invoices {deal_id, subtotal}` body still works (one standard-rated line).

**RERA licences and advertising permits**
- Company: RERA office number (ORN), trade licence number and expiry. Users: broker registration number (BRN) and expiry (`PATCH /users/:id/brn`).
- Listings: advertising permit number, expiry and QR image. In Dubai (Trakheesi) and Abu Dhabi (Madhmoun) a listing needs a valid permit before it can be **published, shared on its public page, emailed in a campaign, or given a QR code**. An admin can allow it anyway with a reason (`POST /listings/:id/permit-override`, audit-logged).
- The public listing page shows the permit number and QR; a listing whose permit has lapsed disappears from it.
- **Expiry alerts** 30 and 7 days before (and once after) a listing permit, BRN or trade licence expires, to the person concerned and the company's admins.

**PDPL**
- **Consent per channel** (WhatsApp, email, SMS, phone) with full history: recorded from the contact page, when creating a contact, from the public lead form (`"consent": {"whatsapp": true}`) and from CSV imports (`consent_whatsapp`, `consent_email`, …).
- **Opt-outs are enforced**: WhatsApp messages, templates and media, single emails and listing email campaigns are refused (or skipped) for contacts who opted out.
- **Data-subject export** (`GET /contacts/:id/export`, admin): everything held about a contact as one JSON file.
- **Erasure** (`POST /contacts/:id/erase`, admin, confirmation required): anonymises the contact and removes personal data from conversations, notes, emails and stored attachments; issued invoices are kept as tax records. Audit-logged.
- **Retention**: a daily job deletes bank statements (and their files), documents and payment confirmations whose `retention_until` has passed.

**Fixed**
- Changing a listing's status (`PATCH /listings/:id/status`) and reviewing an approval request always answered 500: both read the user id with the wrong type.
- Approving a listing approval request now publishes the listing (previously nothing did, so with listing approval switched on a listing could never be published). `published_at` is set when a listing is published.
- `PATCH /listings/:id` could change the status directly, bypassing approval; status changes now only go through the status endpoint.
- Creating a listing failed (NOT NULL violation) unless the client sent `portal_sync_status`, which the web form never does.

#### Upgrade notes
- Migration `0067` converts existing invoices: each gets one line for its deal, the deal's currency, the current company details as supplier and the deal's contact as customer. Numbering continues after the highest existing `INV-YYYY-NNNN`.
- Dubai and Abu Dhabi listings that are already published keep their status but cannot be re-published, shared or campaigned until a permit is entered (or an admin overrides).

### Foundations (#83)

Foundations for the UAE-launch roadmap (#82, milestone #83). Includes migrations `0065` and `0066`.

### Added
- **Background jobs.** A scheduler runs inside the API (`internal/jobs`). Each job is claimed in the database before it runs, so with several replicas it still runs once per interval, and a lease stops a slow run from being started twice. Runs that handled something or failed are logged per company for 30 days and listed under **Settings → Background Jobs** (`GET /api/v1/admin/jobs/runs`). Set `JOBS_ENABLED=false` to keep a process from running jobs.
  - **Viewing reminders:** the assigned agent is notified (in-app, live, and by email when mail is configured) up to an hour before a viewing, once. Rescheduling a viewing re-arms its reminder.
  - **Offer expiry:** submitted or under-review offers past `valid_until` become `expired` and the agent is notified.
- **Country rules** (`internal/jurisdiction`): VAT rates and categories, currency and its decimals, timezone, weekend days, regions (emirates) and tax-number format, per country. Only the UAE is defined so far.
- **Company country, emirate, currency and timezone** (Settings → Company). `GET /api/v1/settings/locale` (every role) and `GET /api/v1/settings/jurisdictions`. The web app formats amounts in the company currency and dates in the company timezone instead of hard-coded AED/Asia/Dubai.

### Changed
- **Money columns hold 3 decimals** (`NUMERIC(p+1,3)`), so BHD, KWD and OMR amounts are stored exactly. Existing values are unchanged. Invoices keep 2 decimals until the invoice model is reworked (#84).
- The company VAT number (TRN) is **optional** (businesses below the registration threshold have none); when given it must have 15 digits.

### Fixed
- **Settings → Company never loaded or saved.** The page used field names the API does not have (`company_name`, `address`, `iban`, …) and read a `data` wrapper the API never returns, so the form stayed empty and every save failed. It now uses the API's fields.
- Notifications created without an id (e.g. "viewing scheduled") were all inserted with the nil UUID, so every one after the first failed silently.

### Upgrade notes
- Migration `0066` rewrites the tables that hold money columns (deals, leads, leases, payments, listings, offers, …). On a large database, run it in a maintenance window.
- A company whose stored VAT number is not 15 digits must correct it the next time it saves company settings.

## [v0.3.4] - Unreleased

### BOS24 marketplace integration retired
- **Removed** the BOS24 (BuyOrSell24) marketplace sync, which was a stub in this edition: `POST /webhooks/bos24`, `GET|PATCH /api/v1/settings/bos24-integration`, `POST /api/v1/settings/bos24-integration/register|sync`, the "BOS24 Sync" settings page and the `bos24_integration_settings` table.
- **Imported ids kept, under the generic names.** Migration `0082` moves `leads.bos24_inquiry_id` to `external_source = 'bos24'` + `external_ref`, and `listings.bos24_listing_uuid` to the same new pair on listings (unique per company; portal publishing will use it). Its down migration restores the old columns.
- New leads can no longer be created with source `bos24` (existing ones keep it).
- **Not changed yet:** the market-data endpoints (`/api/v1/properties/*`, Properties and Map pages) still run on the stub client and answer `503`; a dldapi client will replace it.
- **README** corrected: Go 1.26, Docker Compose instead of "single binary", the market-data section describes the stub, no "resets every 24 hours" or "immutable" claims.
- **Market data is now credited to Waqov** (https://waqov.com/) instead of BuyOrSell24: settings page, DEVELOPER.md and developer page, API doc comments. `BOS24_BASE_URL` now defaults to `https://dldapi.waqov.com`. The `BOS24_*` env var names are unchanged so existing configs keep working.

### Lead intake hardening (`POST /webhooks/leads`)
- **Idempotent pushes.** Optional `external_source` + `external_ref` identify a lead in the system that sends it. The same pair again returns the existing lead (`200`, `"duplicate": true`) instead of creating a second one, so a sender can retry safely. Unique per company. Adds migration `0079`.
- **New source `portal`** for leads from listing portals and marketplaces; it requires `external_source`.
- **Breaking:** an unknown `source` is now rejected with `422` instead of silently stored as `web`.
- **Suspended companies.** An API key whose company is suspended now gets `403 account_suspended` (previously the key kept working).
- **One de-duplication for pushed leads.** Portal webhooks (`/webhooks/portals/:portal`) now record the portal's lead id on the lead (`external_source` = the portal, `external_ref` = its id) instead of the separate `portal_leads` claim table, which is dropped after its rows are copied onto their leads (migration `0080`). A redelivered portal lead returns `{"duplicate": true, "lead_id": …}`.
- **Audit log.** Leads created by API key are now audited, with the key recorded in the new `audit_logs.api_key_id` (previously API-key actions wrote no audit entry at all).

### Security / bug / performance audit
- **Security — email header injection and sender spoofing.** `to_email`/`subject` were written straight into the SMTP headers, so a CR/LF could smuggle `Bcc:` headers or body content, and `POST /emails/send` accepted a client-chosen `from_email`. Recipients are now validated, header values containing CR/LF are refused, the sender is always the configured account, and subjects are RFC 2047-encoded (Arabic subjects no longer arrive garbled).
- **Security — HTML injection in listing email campaigns.** Listing title, area, reference, company name and the free-text message were interpolated unescaped into an email sent to third parties; they are now escaped (cover images must be http(s)).
- **Security — public e-signature.** A signature link could be used repeatedly (overwriting `signed_at`), could complete a DocuSign-managed signature, recorded the *sender's* IP/user agent instead of the signer's, and never marked the document signed. A signature now signs once, only when pending, records the signer's IP/UA, and completes the document when every signature is in. The public GET no longer returns the sender's IP/UA or the DocuSign envelope id.
- **Security — public listing page** no longer exposes `owner_name`, portal sync state or the assigned agent id.
- **Security — password-reset timing.** The reset email was sent synchronously only for registered addresses, revealing account existence by response time; it is now sent in the background.
- **Security** — unauthenticated `/api/public/*` routes are rate limited; download filenames built from user data (`brochure`, `qr`, property report) are sanitised; the websocket caps client frames and reaps dead connections (ping/pong deadlines).
- **Bug** — listing email campaign panicked (nil dereference) when the company had no settings row.
- **Bug** — accepting an offer was check-then-act, so concurrent requests created duplicate deals; acceptance is now claimed atomically (and restored if deal creation fails).
- **Bug** — `GET /message-templates?page=0` returned 500 (negative `OFFSET`).
- **Performance** — the trial/suspension check hit the database on every authenticated request; company status is now cached for 15 s (a suspension or plan change takes effect within that window).
- **Kanban board is bounded.** `GET /leads` loaded every non-deleted lead on each visit. It now returns the newest 100 per stage (`?per_stage=`, max 500), the new `GET /leads/board` adds the true count and deal value of every stage, and `GET /leads/search` accepts a `before`/`before_id` keyset cursor so a column can "Load more" without skipping or repeating cards. The pipeline page shows correct column totals and a "Load more" button. Adds migration `0064` (index on `company_id, stage, created_at`).
- **Bank statements are actually stored.** Upload used to record metadata and discard the file. Files are now kept under `BANK_STATEMENT_DIR` (default `./data/bank-statements`, on the Docker `/app/data` volume; `BANK_STATEMENT_MAX_MB`, default 10) under random names, content-checked against the extension, and served only through the new authenticated, company-scoped `GET /bank-statements/:id/download` (forced download, `nosniff`). Deleting a statement removes the file and keeps the audit row. Adds migration `0063`. Also fixes `GET /bank-statements` (list and detail) failing with a scan error once any statement existed (`import_error` is NULL), and `upload_date` being saved as year 0001.
- **QR codes no longer trust the `Host` header.** The listing QR now encodes `APP_URL/l/<id>` instead of whatever host the request claimed (the old URL also pointed at the API rather than the public page).
- **Outbound webhooks use a bounded worker pool** (8 workers, 1024-event queue; overflow is dropped and logged) instead of a goroutine per event, serialise the payload at dispatch time, cancel retries on timeout, reuse connections, and drain in-flight deliveries on shutdown.
- **Login / sign-up fixes.**
  - **Password reset and invite links never worked from the web app.** The reset page sent `password` but the API reads `new_password`, so every attempt failed validation. (A reset also now clears the failed-login lockout, so the new password works immediately.)
  - **Sign-up always failed when Cloudflare Turnstile was enabled on both web and API.** The page redeemed the single-use token itself before the API did, so the API's own check got "duplicate". The API now does the only verification.
  - **Registering an email that differed only in case/whitespace** passed the uniqueness check, created a company, then failed with a 500 and left that company behind (and mixed-case input was never validated as an email). Sign-up is now one transaction under an advisory lock: emails and subdomains are normalised and validated (3-40 chars, `a-z0-9-`, reserved names refused), duplicates are a clean 409, concurrent sign-ups cannot both become "first user", and nothing is left behind on failure. Passwords over 72 bytes are a 400 instead of a 500, and a Redis failure storing the refresh token is no longer ignored.
  - **Account enumeration via lockout.** The "account locked" response (429) only ever appeared for registered emails; unknown emails are now counted and locked the same way.
  - **SMS OTP** is only sent to registered, active numbers (it used to text any number: SMS-pumping cost and spam), and the per-phone/per-email limits for SMS, magic link and password-reset email are now atomic (concurrent requests could all pass a read-then-increment check).
  - **Magic-link and SMS sign-in** reported `days_remaining: 0` (hiding the trial banner), ignored suspended workspaces and expired trials; they now use the same company checks as password login, and `/auth/refresh` returns the company so plan/trial state no longer goes stale. `/auth/refresh` also now returns the user's phone.
  - Removed `UserRepo.InviteUser` (unused, and it could not have worked: it inserted a token for the nil user id).
- **Lead audit.**
  - **Lead rotation never assigned anything.** `AssignNewLead` was never called, so enabling round-robin/capacity only affected the manual "auto-assign" button. New leads from the API, the public intake endpoint, CSV import and AI auto-create are now assigned when rotation is on.
  - **`PATCH /settings/lead-rotation` reset what it did not mention** (`{"mode":"capacity"}` disabled rotation and zeroed the cap); absent fields now keep their value, a negative cap is refused, and saving no longer rolls the round-robin counter back. Auto-assign reports "rotation not enabled" / "no active agents" instead of "upstream service unavailable", and no longer consumes an agent's turn for a lead that does not exist.
  - **Stage / notes / assign changes on a missing, deleted or other-company lead returned 200**, and a stage move still fired the `lead.stage_changed`/`lead.won`/`lead.lost` webhooks, WebSocket event and scoring for it. They now return 404 and fire nothing. Assigning to a viewer, an inactive user or another company's user is a 422 instead of a silent no-op.
  - **Stage moves** are atomic and report the previous stage: moving a lead to the stage it is already in is a no-op (no duplicate webhooks, `closed_reason` untouched), a reason is only kept on won/lost stages and is cleared when a lead is reopened, and `lead.won`/`lead.lost` follow the pipeline stage's `is_won`/`is_lost` flags (custom-named closing stages never fired them). Stage changes and assignments are now audit-logged and assignments are broadcast to other users.
  - **Creating leads**: a new lead starts in the company's default pipeline stage (and must use a real stage), `source` defaults to `web` (an empty source used to fail the DB CHECK), currency must be a 3-letter code (longer ones were a 500) and is upper-cased, `deal_value` must be 0–9,999,999,999.99, notes are capped at 10,000 characters.
  - **Public intake (`POST /webhooks/leads`)**: every submission reset an existing contact's language to Arabic (the default was always "provided") and could overwrite its email; it now only fills a missing email and only applies a language the caller actually sent (validated `ar`/`en`, email format checked). A retry or double-submit within 10 minutes returns the existing lead (`200`, `duplicate: true`) instead of creating another. Currency/deal value are validated like the main API.
  - **CSV lead import** validated rows after creating their contact (rejected rows left orphan contacts), silently turned an unparsable `deal_value` into 0 and passed bad currencies through to a 500; it now validates the whole row first and reports each problem.
  - Searching leads or contacts for `%` or `_` treated them as wildcards (e.g. `%` matched everything); they now match literally. AI auto-create ignores budgets the column cannot hold, and `GET /leads/:id/communications` returns `[]` instead of `null`.
- **Contacts / deals / invoices audit.**
  - **Deleting a contact silently deleted its leads, deals, offers, viewings and WhatsApp conversations** (all `ON DELETE CASCADE`), and a contact whose deals had invoices failed with a confusing 422. `DELETE /contacts/:id` now answers 409 with the counts of what would go unless `?force=true`, always refuses when invoices exist, and returns 404 for an unknown id (it said 204). The contact page asks for a second confirmation listing the records.
  - **A deal with a close date could not be edited from the web app.** `GET` returns `close_date` as a timestamp, the page sent it straight back, and `PATCH` only accepted `YYYY-MM-DD` (400). Both formats are accepted, `""` clears the date, and the page now uses the date part.
  - **Contact PATCH could not clear an email or unassign a contact** (empty/`null` were treated as "not provided"); both work now. Contacts validate name, email, language (`ar`/`en`, the column is `varchar(2)` so other values were a 500) and lead score (0–100); assignees must be active admins/agents; client-supplied ids/timestamps are ignored; a legacy invalid value no longer blocks an unrelated edit.
  - **Deals**: probability can be 0 (it was forced to 50), won/lost stages set probability to 100/0 and a close date, so weighted forecasts are right; stage/delete of a missing deal is 404 (was 200/204), deleting a deal with invoices is a clear 409, invalid `owner_id`/`stage` filters are 400, title/amount/currency are validated, deals cannot be added to a deleted lead, and the deal's invoice list is `[]` (not `null`) and 404 for an unknown deal.
  - **Invoice numbers** (`INV-YYYY-NNNN`): the number was computed in one query and inserted in another, so two invoices created together got the same number and one failed; and the next number was read from the last four characters, so the 10,000th invoice of a year became `1000` (duplicate) and any unparsable number broke numbering. Numbering and insert now happen in one transaction under a per-company lock, and 5+ digit numbers work. Sending an invoice only works from draft (re-sending a paid invoice flipped it back to "sent"), unknown invoices are 404, subtotals are rounded to fils and bounded, and PDF filenames are sanitised.
- **Listings / properties / approvals audit.**
  - **Changing a listing's status always failed (500)** and **no approval request could ever be reviewed (500)**: both handlers read the user id from the request as a string, but it is a `uuid.UUID`, so they panicked. The whole publish/approve workflow was unusable.
  - **Approving a request never published the listing** (it only flipped the request). Approval now publishes the listing in the same transaction; a request can be reviewed once (409 afterwards, 404 if unknown), and asking to publish again while one is pending reuses it instead of piling up duplicates.
  - **Approval bypass.** `PATCH /listings/:id` and `POST /listings` accepted `status` from the client, so a listing could be published without approval. Status now changes only through `/listings/:id/status`; creating with `status: published` publishes only when no approval is required.
  - **Creating a listing failed with "invalid data"** unless the client sent `portal_sync_status` (NOT NULL jsonb sent as NULL), and **editing one failed to parse** whenever the date field was empty (the form sends `available_from: ""`). `published_at` was never set; it now is. The body can no longer overwrite the listing's id (which redirected the update to another listing), company or audit fields.
  - **Validation** for listings (type, listing type, price > 0, currency, coordinates, counts, lengths) and rental properties (type, status, occupancy, units/occupied, money, lengths). Rental properties stored zero values literally (status `''`, currency `''`, 0 units) because the INSERT bypassed the column defaults; defaults are now applied. Updates now persist currency, purchase price and purchase date (the response claimed they were saved). Over-long values anywhere are a 422 instead of a 500.
  - **Deleting** a listing silently deleted its offers (cascade); it now needs `?force=true` when offers exist (the listing page asks). Deleting a rental property that has leases or expenses is a clear 409 instead of a confusing 422; unknown ids are 404 instead of 204. Listing assignees must be active admins/agents. The listing page now says "sent for approval" instead of "status updated" when publishing needs approval.
- **Tenants, leases, payments and renewals audit.** Adds migration `0081`.
  - **Leases could not be read back.** `termination_reason` is NULL until a lease is edited and was scanned into a plain string, so `GET /leases/:id` (and therefore every `PATCH`, the lease page, renewals, payment confirmations) failed with "lease not found" for every freshly created lease. Nullable text columns are now coalesced.
  - **Tenants: verification cannot be forged.** `is_verified`/`verification_*` were taken from the create/update body, so any agent could create or edit a tenant as "verified"; only `POST /tenants/:id/verify` sets them (and now stamps `verification_date`). `id_number` uniqueness was global, so one company's tenant blocked another company's; it is now unique per company (a duplicate is a 409), as are bank-transaction external ids. ID type, employment and status are validated (both the web form's and the API's vocabulary are accepted); empty date inputs no longer fail to parse; deleting a missing or referenced tenant is a 404/409 instead of a silent 204/500.
  - **Leases: validation and double-booking.** End must follow start (max 20 years), rent must be positive, fees/percentages/day-of-month/notice are range-checked, frequency and status are validated (a lease can only be created `active`), and an active lease overlapping another on a single-unit property is a 409 (also on re-activation). Editing a lease no longer rewrites its property, tenant, term or currency (the response used to echo values that were never saved); terminating stamps `termination_date`; a body `id` can no longer redirect an update. Leases that have payments cannot be deleted (409, terminate instead).
  - **Payments.** `GET /payments?lease_id=` filters on the server (the lease page used to fetch the first 100 payments of the whole company and filter in the browser, so older leases looked empty). Amounts must be positive, dates/method/status are validated, a "received" payment gets `paid_date`, the currency always follows the lease, and the reconciliation fields (`bank_transaction_id`, `reconciled_*`) are no longer client-writable. Updates cannot move a payment to another lease.
  - **Lease templates.** Validated like leases, only one default per company (a new default demotes the old one; `GetDefault` is deterministic), ids in the body are ignored, deleting an in-use template is a 409 and a missing one a 404.
  - **Renewals.** The renewal date is the lease's end date (it was "90 days from now"), only active leases can be renewed, starting a second renewal is a 409, accepted/rejected renewals are final, proposed and counter amounts must be positive, and sending an offer without a template no longer marks the renewal "offer sent" and then fails on a zero-UUID foreign key.
- **Payments, invoices and bank integrations audit.**
  - **Invoices could not be read back.** `qr_payload` is NULL on every new invoice and was scanned into a plain string, so `GET /invoices`, `GET /invoices/:id`, the invoice PDF and a deal's invoice list all answered 500 as soon as one invoice existed. Fixed (the invoices page was effectively empty).
  - **Invoice status.** A sent or paid invoice can no longer be set back to draft (409), `?status=` now filters on the server (the page filtered only the 50 rows it had fetched), lookups of a missing invoice are 404 and other errors are no longer reported as "not found", and the PDF endpoint answers 501 in this edition instead of a generic 500.
  - **Bank integrations could not be read back either** (`last_sync_error` NULL), and they accepted anything: empty status/type, a zero sync interval, plain-http endpoints, junk IBANs. They are now validated (type manual/api/file_import, status, 1-720 h interval, https endpoint, IBAN shape), `is_connected`, sync state and an `id` in the body are ignored on update (the id used to redirect the update to another integration), and deleting an integration that has statements is a 409 (it used to cascade-delete the uploaded statements and orphan their files); a missing one is a 404.
  - **Financial analytics always failed.** `GET /analytics/financial` selected `expenses.category_type`, a column that does not exist; it now joins the category, ignores soft-deleted expenses and also returns `rent_overdue`.
  - **Overdue rent was always zero.** Nothing ever sets a payment's status to `overdue`, so the dashboard counts and amounts were 0; a pending payment past its due date now counts as overdue (and no longer as pending) in the tenant and financial analytics.
  - **Payments.** A payment matched to a bank transaction can no longer have its amount or status changed or be deleted (409); a pending/overdue payment drops any paid date; the payment-method error lists the accepted values.
  - **Payment confirmations** answered 201 with `null` when nothing was sent (the community edition's service is a no-op). They now check the payment exists in the company (404) and is received (409), and answer 501 when the edition cannot send one.
- **Expenses, maintenance and inspections audit.**
  - **Expenses could not be created from the web app.** The form posts a date-only `expense_date` and `""` for "no property", neither of which the handler could decode, so every create answered 400. Both are accepted now; the amount (> 0, whole fils), description, date, payment method/status and receipt URL (http(s) only) are validated, categories and properties must belong to the company (422, was 500), and a duplicate category name is a 409.
  - **Expense edits** can now change more than amount/description/status (category, date, vendor, method, receipt, notes); an approved expense's amount is locked; approving a missing expense is a 404, approving twice a 409 (was 500) and the web client's body-less approve works (it used to be "invalid request body"); deleting a missing expense is a 404.
  - **Maintenance tasks.** Date-only dates and `""` references are accepted (the DATE columns could not be filled from a form); type, priority, status, description and costs are validated; a missing property/assignee is a 422 (was 500); completing through `PATCH` sets the completion date and re-opening clears it; a task can be completed only once and a closed task is not editable unless re-opened; `POST .../complete` with no body (what the web client sends without a cost) no longer answers 400; the list sorts by real priority (urgent, high, medium, low) instead of alphabetically; deleting a missing task is a 404.
  - **Photo links** (maintenance photos, inspection photos) must be http(s) or site-relative: a `javascript:`/`data:` URL used to be stored and rendered as a link. Photos can only be added to (and read from) an existing task.
  - **Inspections.** Type, status, severity, schedule date and checklist results are validated, optional references may be `""`, a missing property/template/inspector is a 422 and a missing inspection a 404 (both were 500), completing twice no longer rewrites the completion time (409), and an empty `photos_urls` clears the photos. Template names are unique (409) and templates are validated (type, duration, checklist items).
  - **Properties with maintenance tasks or inspections can't be deleted** (409); neither table has a foreign key to the property, so they were left orphaned.
  - **`GET /analytics/maintenance` returned the wrong numbers** (average completion days and the high-priority count were scanned from each other's columns, and the average into an int, which fails for fractional values); it now counts open high-priority work and ignores soft-deleted tasks.
- **Performance / robustness** — SMTP connections have connect and overall timeouts (a stalled relay used to hang the request forever); `last_used_at` on API keys is written at most once a minute instead of on every call; `GET /leads/search` (`limit` ≤ 200) and `GET /leads/:id/communications` (`limit` ≤ 500) can no longer be asked for an unbounded result.

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
