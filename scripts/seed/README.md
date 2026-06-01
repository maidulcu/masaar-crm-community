# Seeder — Demo Data for Masaar CRM

Seeds all 38 database tables with realistic UAE demo data for evaluation and development.

## Quick Start

```bash
go run ./scripts/seed
```

Uses `DATABASE_URL` from environment, or defaults to `postgres://masaar:masaar@localhost:5432/masaar?sslmode=disable`.

## Demo Accounts

| Email | Password | Role | Language |
|-------|----------|------|----------|
| ahmed@masaar.local | Demo@1234 | admin | Arabic |
| sarah@masaar.local | Demo@1234 | agent | English |
| mohammed@masaar.local | Demo@1234 | agent | Arabic |
| lisa@masaar.local | Demo@1234 | viewer | English |

## What Gets Seeded

| Entity | Rows | Notes |
|--------|------|-------|
| Company | 1 | Masaar Properties Demo — 25 days remaining on trial |
| Users | 4 | Admin + 2 agents + 1 viewer |
| Contacts | 10 | UAE contacts with Arabic/English names, +97150 numbers |
| Leads | 10 | All pipeline stages: new → won/lost |
| Lead Tags | 6 | budget-conscious, vip, premium, etc. |
| Communication History | 6 | WhatsApp + email inbound/outbound |
| Deals | 5 | Mix of open / won / lost (AED 29K–550K) |
| VAT Invoices | 4 | Draft / sent / paid with 5% UAE VAT |
| WhatsApp Threads | 5 | Arabic + English conversations |
| WhatsApp Messages | 18 | 2–5 messages per thread |
| WhatsApp Outbound | 3 | Sent outbound records |
| Notifications | 6 | New lead, deal won, lease expiring, overdue payment |
| Rental Properties | 8 | Dubai Marina, Downtown, Palm, DIFC, JLT, etc. |
| Tenants | 8 | UAE residents, verified Emirates IDs |
| Lease Templates | 4 | Standard, Premium Villa, Commercial, Short-Term |
| Leases | 7 | Active / terminated across properties |
| Payments | 12 | Paid / pending / overdue — AED 6.5K–45K |
| Expense Categories | 6 | Utilities, Maintenance, Marketing, Insurance, etc. |
| Expenses | 9 | DEWA bills, marketing, insurance, legal retainer |
| Inspection Templates | 3 | Move-In, Move-Out, Quarterly Audit |
| Inspections | 3 | Completed / scheduled |
| Maintenance Tasks | 5 | AC repair, pool pump, plumbing, painting, elevator |
| Lease Renewals | 2 | Negotiating / pending |
| Renewal Comm Templates | 2 | English + Arabic |
| Renewal Comm Log | 2 | Email + WhatsApp delivery records |
| Commission Structures | 2 | Sales + Leasing |
| Agent Commissions | 2 | Approved / pending |
| Document Templates | 3 | Lease agreements (EN/AR) + maintenance request |
| Documents | 3 | Signed / pending / N/A |
| Document Signatures | 1 | Signed by John Smith |
| Message Templates | 5 | Welcome, viewing confirmation, payment reminder |
| Custom Field Definitions | 3 | preferred_contact_time, budget_range, pet_friendly |
| API Keys | 2 | Production + Webhook |
| Webhook Subscriptions | 2 | Lead sync + payment notifications |
| Bank Integrations | 1 | Emirates NBD |
| Bank Transactions | 4 | Rent payments received |
| Bank Statements | 1 | Processed CSV |
| Payment Confirmations | 1 | Delivered via email |
| Usage Tracking | 7 | Contact/lead/deal/property/tenant/lease/message counts |
| Audit Logs | 5 | Login, stage change, deal created, payment received |
| Company Settings | 1 | VAT, bank details, business address |
| API Settings | 5 | Business hours, currency, lead assignment, etc. |
| Listings | 8 | Published + draft property listings in Dubai/Abu Dhabi |
| Offers | 3 | Submitted / countered / accepted |
| Viewings | 4 | Completed / scheduled / confirmed |
| Pipeline Stages | 6 | Default lead stages (new → won/lost) |
| Lead Rotation Settings | 1 | Round-robin enabled |
| Agent Targets | 4 | Monthly targets for Sarah + Mohammed |
| Approval Configs | 1 | Listing approval on, deal/offer thresholds |

## Idempotency

The script uses **deterministic UUIDs** (`uuid.NewSHA1`) — same seed key always produces the same UUID. Combined with `ON CONFLICT DO NOTHING` on natural unique keys, re-running the script is safe and will not duplicate data.

## Schema Compatibility

Requires migration 0045 (`saas_multi_tenancy`) to have been applied. The seed creates data under a single demo company with `company_id` correctly set on all scoped tables.

## Future

To reset demo data:
```sql
TRUNCATE ...  -- cascade all tables
go run ./scripts/seed
```
