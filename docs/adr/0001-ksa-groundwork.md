# ADR 0001: Saudi Arabia groundwork — what the community edition covers

- **Status:** proposed
- **Context issue:** #88 (M5), tracking #82

> **Not legal advice.** The Saudi rules summarised here (ZATCA e-invoicing, REGA
> advertising licences, the PDPL) change; check the official sources before acting on them.

## Context

After the UAE, the next target market is Saudi Arabia (then the rest of the GCC). M0 made
the country a per-company setting (`internal/jurisdiction`), so the question is what else
a Saudi brokerage needs and which parts belong in this community repository.

## Decision

### In the community edition now

- **Jurisdiction `SA`**: VAT 15 %, SAR, `Asia/Riyadh`, Friday–Saturday weekend, 15-digit
  VAT number, the 13 administrative regions. Choosing Saudi Arabia in Settings → Company
  changes invoice VAT, currency and job timing like any other country.
- **REGA advertising licence**: every Saudi region requires a permit on listings
  (`ListingPermit: "rega"`), enforced the same way as Trakheesi/Madhmoun in the UAE (a
  listing cannot be published or shared without a valid permit unless an admin overrides).
- **Hijri dates**: an optional company setting shows the Umm al-Qura date next to dates in
  the web app. Dates are still stored and entered as Gregorian.
- **A provider-neutral e-invoicing interface** (`einvoice.Connector`), built for the UAE
  but not tied to it.

### Not in the community edition

- **ZATCA e-invoicing (Fatoora), phase 2.** It needs a UBL 2.1 profile of its own, an
  X.509 cryptographic stamp per invoice, a hash chain between invoices, a TLV QR code, and
  device onboarding (CSID). Standard (B2B) invoices are *cleared* by ZATCA before they are
  issued; simplified (B2C) invoices are *reported* within 24 hours. ZATCA e-invoicing is an
  Enterprise edition feature (see the README). A ZATCA implementation could reuse the
  `Connector` interface, but clearance changes the flow: the invoice is not issued until
  ZATCA answers, whereas PINT AE submits an already-issued invoice.
- **Ejar** (rental contract registration) and **Nafath** (national identity sign-in): both
  require an agreement with the operator and credentials that a self-hosted community
  install cannot obtain on its own. Out of scope. A lease's registration number field
  (labelled Ejari for the UAE) can hold an Ejar contract number entered by hand.

### Data residency

The Saudi PDPL restricts transfers of personal data outside the Kingdom. Masaar CRM is
self-hosted: the operator chooses where the database, file uploads and backups live.
Saudi operators should:

- host the database, `UPLOAD_DIR` and backups in the Kingdom;
- review the optional outbound integrations (email provider, WhatsApp Cloud API, portal
  webhooks, AI features) for where they process data;
- use the existing PDPL tools (consent, export, erasure, retention) — they are country
  neutral.

No code change is needed for residency; it is a deployment choice, documented here.

## Consequences

- Saudi companies can run the CRM with correct VAT, currency, weekend and listing rules.
- Tax invoices for Saudi companies are printable tax invoices only; they are not ZATCA
  e-invoices and must not be presented as such.
- A future ZATCA module plugs in behind `Connector` with a clearance step before issue.
