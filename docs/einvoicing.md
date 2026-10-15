# UAE e-invoicing (PINT AE)

> **Not legal or tax advice.** The UAE e-invoicing system is being phased in by the
> Ministry of Finance and the Federal Tax Authority. Check the scope, the timeline and the
> data requirements that apply to your business with the official sources and with your
> accredited service provider (ASP) before relying on this feature.

Masaar CRM can turn a tax invoice or credit note into a **PINT AE** e-invoice (the UAE
specialisation of the Peppol international invoice, UBL 2.1) and hand it to an ASP through
a small connector interface. It is off by default: invoices stay PDF/print only until an
admin picks a provider in **Settings → E-invoicing**.

## What is built

| Piece | Where |
|-------|-------|
| UBL 2.1 Invoice / CreditNote in PINT AE (`urn:peppol:pint:billing-1@ae-1`) | `internal/einvoice/pintae.go` |
| Golden files, checked against the official rules in CI | `internal/einvoice/testdata/*.xml`, job *E-invoice (PINT AE)* |
| Provider interface and a mock provider | `internal/einvoice/connector.go` |
| Submissions and their status | table `einvoices` (migration `0078`), `internal/repo/einvoice.go` |
| Status polling and rejection notices | job `einvoice_status` (every 5 minutes), `internal/jobs/einvoice.go` |
| API | `GET/POST /invoices/:id/einvoice`, `POST /invoices/:id/einvoice/xml`, `GET /invoices/:id/einvoice.xml`, `GET/PUT /settings/einvoicing` |
| UI | the *E-invoice* panel on an invoice, Settings → E-invoicing |

### Data the e-invoice needs

Beyond the tax invoice, PINT AE needs:

- **Seller**: the 10-digit **TIN** (the e-invoicing participant ID), the TRN, the legal
  registration (trade licence by default, with its issuing authority), street, city and
  emirate (taken from the company's region), and the IBAN for the payment means.
- **Buyer**: the same set: name, TIN, TRN if VAT-registered, legal registration, address,
  emirate. Entered on the invoice's E-invoice panel and kept with the submission.
- **Invoice**: a due date (30 days after issue unless given).
- **Credit note**: the reason code (`DL8.61.1.A`–`E`, the cases of Article 61(1) of the
  VAT Decree-Law, or `VD` for a volume discount) and the invoice it corrects.
- **Exempt lines**: a VAT exemption reason code (company setting; use the code your ASP
  specifies).

When something is missing, the export and submit endpoints answer `422` with
`{"error": "...", "missing": ["buyer TIN (10 digits, ...)", ...]}`.

### Known limits

- **AED only.** A document in another currency also needs the AED tax total and the
  exchange rate, which the invoice model does not hold yet.
- **One SAC for all lines.** Service lines carry the company's service accounting code.
- **No real ASP connector ships.** The `mock` provider accepts everything, or rejects a
  document whose notes contain `#mock-reject`, so the workflow can be tried end to end.
- **Assumptions to verify with an ASP**: the TIN is entered, not derived from the TRN;
  the exemption reason code is free text; the issuing authority is free text.

## Validating the XML

`scripts/pint-ae/validate.sh file.xml ...` checks documents against the UBL 2.1 schema and
the official PINT AE Schematron (release 2026.5), as packaged in
`com.helger.phive.rules:phive-rules-peppol-pint` on Maven Central. The script downloads the
pinned jars (SHA-256 checked) to `~/.cache/masaar-pint-ae` and needs `java` and `xmllint`.
It fails on any fatal assertion. CI runs it on the golden files.

After changing the generator, regenerate and re-validate the golden files:

```sh
go test ./internal/einvoice/ -update
scripts/pint-ae/validate.sh internal/einvoice/testdata/*.xml
```

Review the diff of `testdata/` like code: it is the contract with the ASP.

## Adding a provider

Implement `einvoice.Connector` and register it in an `init()`:

```go
type Connector interface {
	Name() string
	Submit(ctx context.Context, s Submission) (ref string, err error)
	Status(ctx context.Context, ref string) (Result, error)
}

func init() { einvoice.Register(myASP{...}) }
```

- `Submit` sends the XML and returns the provider's reference. An error is recorded as
  status `error` (the invoice can be submitted again).
- `Status` reports `submitted` (still processing), `accepted` or `rejected` with a reason.
  The `einvoice_status` job asks every five minutes; on rejection the issuer and the
  submitter are notified.
- Keep credentials in environment variables, never in the database or the repository.
