// Package einvoice builds UAE electronic invoices in the Peppol PINT AE format (UBL 2.1)
// and hands them to an accredited service provider through a Connector.
//
// The XML is checked against the official UBL schema and the PINT AE Schematron rules by
// scripts/pint-ae/validate.sh (also run in CI). The format and the rollout dates are set by
// the UAE Ministry of Finance and the FTA; check the current specification before relying
// on this in production.
package einvoice

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// PINT AE identifiers (specification identifier and business process).
const (
	CustomizationID = "urn:peppol:pint:billing-1@ae-1"
	ProfileID       = "urn:peppol:bis:billing"
	EndpointScheme  = "0235" // UAE TIN
)

// Document kinds.
const (
	KindInvoice    = "invoice"
	KindCreditNote = "credit_note"
)

// Legal registration identifier types (BTAE-15/16).
const (
	LegalTradeLicence = "TL"
	LegalEmiratesID   = "EID"
	LegalPassport     = "PAS"
	LegalCabinet      = "CD"
)

// Emirates are identified by these codes in addresses.
var Emirates = map[string]string{
	"AE-AZ": "AUH", "AE-DU": "DXB", "AE-SH": "SHJ", "AE-UQ": "UAQ", "AE-FU": "FUJ", "AE-AJ": "AJM", "AE-RK": "RAK",
}

var emirateNames = map[string]string{
	"AUH": "Abu Dhabi", "DXB": "Dubai", "SHJ": "Sharjah", "UAQ": "Umm Al Quwain", "FUJ": "Fujairah", "AJM": "Ajman", "RAK": "Ras Al Khaimah",
}

// Party is a seller or a buyer.
type Party struct {
	Name           string `json:"name"`
	TIN            string `json:"tin"`             // 10-digit tax identification number: the Peppol endpoint
	TRN            string `json:"trn"`             // 15-digit VAT registration number ("" if not registered)
	LegalID        string `json:"legal_id"`        // trade licence, Emirates ID, passport or cabinet decision number
	LegalIDType    string `json:"legal_id_type"`   // TL | EID | PAS | CD
	LegalAuthority string `json:"legal_authority"` // licensing authority (TL) or passport country (PAS)
	Street         string `json:"street"`
	City           string `json:"city"`
	Emirate        string `json:"emirate"` // AUH, DXB, SHJ, UAQ, FUJ, AJM, RAK
	Country        string `json:"country"` // ISO 3166-1 alpha-2
	Email          string `json:"email,omitempty"`
}

// Line is one invoice line.
type Line struct {
	Description string
	Quantity    float64
	UnitPrice   float64
	Net         float64
	Category    string  // standard | zero_rated | exempt | out_of_scope
	Rate        float64 // fraction, e.g. 0.05
	VAT         float64
}

// Document is an invoice or credit note to convert.
type Document struct {
	Kind       string
	Number     string
	UUID       string
	IssueDate  time.Time
	DueDate    *time.Time
	Currency   string
	Seller     Party
	Buyer      Party
	Lines      []Line
	Note       string
	IBAN       string // seller's account, for payment by credit transfer
	ServiceSAC string // service accounting code for the lines (brokerage services)
	// ExemptionCode is the VAT exemption reason code (IBT-186) for exempt lines, from the
	// PINT AE code list; required when the document has an exempt line.
	ExemptionCode string
	// Credit notes: the invoice credited and the reason code (BTAE-03).
	PrecedingNumber string
	PrecedingDate   time.Time
	CreditReason    string
}

// MissingError lists the data an e-invoice needs that is not on file.
type MissingError struct{ Fields []string }

func (e *MissingError) Error() string {
	return "missing for the e-invoice: " + strings.Join(e.Fields, ", ")
}

func digits(s string, n int) bool {
	if len(s) != n {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func checkParty(p Party, who string, seller bool) []string {
	var m []string
	add := func(f string) { m = append(m, who+" "+f) }
	if strings.TrimSpace(p.Name) == "" {
		add("name")
	}
	if !digits(p.TIN, 10) {
		add("TIN (10 digits, the e-invoicing participant ID)")
	}
	if (seller || p.TRN != "") && !digits(p.TRN, 15) {
		add("TRN (15 digits)")
	}
	if p.LegalID == "" {
		add("legal registration number (trade licence, Emirates ID or passport)")
	}
	switch p.LegalIDType {
	case LegalTradeLicence, LegalPassport:
		if p.LegalAuthority == "" {
			add("licensing authority or passport country")
		}
	case LegalEmiratesID, LegalCabinet:
	default:
		add("legal registration type (TL, EID, PAS or CD)")
	}
	if p.Street == "" || p.City == "" {
		add("street and city")
	}
	if p.Country == "" {
		add("country")
	}
	if p.Country == "AE" && emirateNames[p.Emirate] == "" {
		add("emirate")
	}
	return m
}

// Check reports the data the document lacks.
func (d *Document) Check() error {
	var m []string
	m = append(m, checkParty(d.Seller, "seller", true)...)
	m = append(m, checkParty(d.Buyer, "buyer", false)...)
	if d.Number == "" || d.UUID == "" {
		m = append(m, "number")
	}
	if len(d.Lines) == 0 {
		m = append(m, "lines")
	}
	for _, l := range d.Lines {
		if l.Category == "exempt" && d.ExemptionCode == "" {
			m = append(m, "VAT exemption reason code (for the exempt lines)")
			break
		}
	}
	if d.Currency != "" && d.Currency != "AED" {
		m = append(m, "AED currency (invoices in other currencies need the AED exchange rate, not supported yet)")
	}
	if d.Kind == KindInvoice && d.DueDate == nil {
		m = append(m, "payment due date")
	}
	if d.Kind == KindInvoice && d.IBAN == "" {
		m = append(m, "seller bank account (IBAN)")
	}
	if d.Kind == KindCreditNote && (d.PrecedingNumber == "" || d.CreditReason == "") {
		m = append(m, "credited invoice and reason")
	}
	if len(m) > 0 {
		return &MissingError{Fields: m}
	}
	return nil
}

// category codes (UNCL5305 subset used by PINT AE).
func categoryCode(c string) string {
	switch c {
	case "standard":
		return "S"
	case "zero_rated":
		return "Z"
	case "exempt":
		return "E"
	default:
		return "O"
	}
}

// ── XML ──────────────────────────────────────────────────────────────────────

type node struct {
	name  string
	attrs [][2]string
	text  string
	kids  []*node
}

func el(name string, kids ...*node) *node { return &node{name: name, kids: kids} }
func tx(name, text string, attrs ...string) *node {
	n := &node{name: name, text: text}
	for i := 0; i+1 < len(attrs); i += 2 {
		n.attrs = append(n.attrs, [2]string{attrs[i], attrs[i+1]})
	}
	return n
}

var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

func (n *node) write(b *strings.Builder, depth int) {
	if n == nil {
		return
	}
	ind := strings.Repeat("  ", depth)
	b.WriteString(ind + "<" + n.name)
	for _, a := range n.attrs {
		b.WriteString(" " + a[0] + `="` + escaper.Replace(a[1]) + `"`)
	}
	if len(n.kids) == 0 {
		b.WriteString(">" + escaper.Replace(n.text) + "</" + n.name + ">\n")
		return
	}
	b.WriteString(">\n")
	for _, k := range n.kids {
		k.write(b, depth+1)
	}
	b.WriteString(ind + "</" + n.name + ">\n")
}

func amt(v float64) string {
	s := fmt.Sprintf("%.2f", math.Round(v*100)/100)
	if s == "-0.00" {
		return "0.00"
	}
	return s
}

func pct(rate float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", rate*100), "0"), ".")
}

func qty(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", v), "0"), ".")
}

func (p Party) xml(role string) *node {
	party := el("cac:Party",
		tx("cbc:EndpointID", p.TIN, "schemeID", EndpointScheme),
	)
	addr := el("cac:PostalAddress",
		tx("cbc:StreetName", p.Street),
		tx("cbc:CityName", p.City),
	)
	if p.Emirate != "" {
		addr.kids = append(addr.kids, tx("cbc:CountrySubentity", p.Emirate))
	}
	addr.kids = append(addr.kids, el("cac:Country", tx("cbc:IdentificationCode", p.Country)))
	party.kids = append(party.kids, addr)
	if p.TRN != "" {
		party.kids = append(party.kids, el("cac:PartyTaxScheme",
			tx("cbc:CompanyID", p.TRN),
			el("cac:TaxScheme", tx("cbc:ID", "VAT"))))
	}
	legalID := tx("cbc:CompanyID", p.LegalID, "schemeAgencyID", p.LegalIDType)
	if p.LegalAuthority != "" {
		legalID.attrs = append(legalID.attrs, [2]string{"schemeAgencyName", p.LegalAuthority})
	}
	party.kids = append(party.kids, el("cac:PartyLegalEntity", tx("cbc:RegistrationName", p.Name), legalID))
	if p.Email != "" {
		party.kids = append(party.kids, el("cac:Contact", tx("cbc:ElectronicMail", p.Email)))
	}
	return el(role, party)
}

type taxGroup struct {
	code    string
	rate    float64
	taxable float64
	tax     float64
}

// XML renders the document as PINT AE UBL. Call Check first.
func (d *Document) XML() ([]byte, error) {
	if err := d.Check(); err != nil {
		return nil, err
	}
	credit := d.Kind == KindCreditNote
	root, typeCode, lineName, qtyName := "Invoice", "380", "cac:InvoiceLine", "cbc:InvoicedQuantity"
	ns := "urn:oasis:names:specification:ubl:schema:xsd:Invoice-2"
	if credit {
		root, typeCode, lineName, qtyName = "CreditNote", "381", "cac:CreditNoteLine", "cbc:CreditedQuantity"
		ns = "urn:oasis:names:specification:ubl:schema:xsd:CreditNote-2"
	}
	allOutOfScope := true
	for _, l := range d.Lines {
		if l.Category != "out_of_scope" {
			allOutOfScope = false
		}
	}
	if allOutOfScope && !credit {
		typeCode = "480" // out of scope of VAT
	}
	cur := d.Currency
	if cur == "" {
		cur = "AED"
	}

	doc := &node{name: root, attrs: [][2]string{
		{"xmlns", ns},
		{"xmlns:cac", "urn:oasis:names:specification:ubl:schema:xsd:CommonAggregateComponents-2"},
		{"xmlns:cbc", "urn:oasis:names:specification:ubl:schema:xsd:CommonBasicComponents-2"},
	}}
	add := func(ns ...*node) { doc.kids = append(doc.kids, ns...) }
	add(tx("cbc:CustomizationID", CustomizationID),
		tx("cbc:ProfileID", ProfileID),
		tx("cbc:ProfileExecutionID", "00000000"), // invoice transaction type: none of the special cases
		tx("cbc:ID", d.Number),
		tx("cbc:UUID", d.UUID),
		tx("cbc:IssueDate", d.IssueDate.Format("2006-01-02")))
	if !credit && d.DueDate != nil {
		add(tx("cbc:DueDate", d.DueDate.Format("2006-01-02")))
	}
	if credit {
		add(tx("cbc:CreditNoteTypeCode", typeCode))
	} else {
		add(tx("cbc:InvoiceTypeCode", typeCode))
	}
	if d.Note != "" {
		add(tx("cbc:Note", d.Note))
	}
	add(tx("cbc:DocumentCurrencyCode", cur))
	if credit {
		add(el("cac:DiscrepancyResponse", tx("cbc:ResponseCode", d.CreditReason)))
		add(el("cac:BillingReference", el("cac:InvoiceDocumentReference",
			tx("cbc:ID", d.PrecedingNumber), tx("cbc:IssueDate", d.PrecedingDate.Format("2006-01-02")))))
	}
	add(d.Seller.xml("cac:AccountingSupplierParty"), d.Buyer.xml("cac:AccountingCustomerParty"))
	if !credit {
		add(el("cac:PaymentMeans", tx("cbc:PaymentMeansCode", "30"),
			el("cac:PayeeFinancialAccount", tx("cbc:ID", d.IBAN))))
	}

	// VAT breakdown per category and rate.
	groups := map[string]*taxGroup{}
	var lineTotal, taxTotal float64
	for _, l := range d.Lines {
		code := categoryCode(l.Category)
		key := fmt.Sprintf("%s/%.4f", code, l.Rate)
		g := groups[key]
		if g == nil {
			g = &taxGroup{code: code, rate: l.Rate}
			groups[key] = g
		}
		g.taxable += l.Net
		g.tax += l.VAT
		lineTotal += l.Net
		taxTotal += l.VAT
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tt := el("cac:TaxTotal", tx("cbc:TaxAmount", amt(taxTotal), "currencyID", cur))
	for _, k := range keys {
		g := groups[k]
		cat := el("cac:TaxCategory", tx("cbc:ID", g.code))
		if g.code == "S" || g.code == "Z" {
			cat.kids = append(cat.kids, tx("cbc:Percent", pct(g.rate)))
		}
		if g.code == "E" {
			cat.kids = append(cat.kids, tx("cbc:TaxExemptionReasonCode", d.ExemptionCode),
				tx("cbc:TaxExemptionReason", "Exempt from VAT"))
		}
		if g.code == "O" {
			cat.kids = append(cat.kids, tx("cbc:TaxExemptionReason", "Not subject to VAT"))
		}
		cat.kids = append(cat.kids, el("cac:TaxScheme", tx("cbc:ID", "VAT")))
		tt.kids = append(tt.kids, el("cac:TaxSubtotal",
			tx("cbc:TaxableAmount", amt(g.taxable), "currencyID", cur),
			tx("cbc:TaxAmount", amt(g.tax), "currencyID", cur),
			cat))
	}
	add(tt)
	add(el("cac:LegalMonetaryTotal",
		tx("cbc:LineExtensionAmount", amt(lineTotal), "currencyID", cur),
		tx("cbc:TaxExclusiveAmount", amt(lineTotal), "currencyID", cur),
		tx("cbc:TaxInclusiveAmount", amt(lineTotal+taxTotal), "currencyID", cur),
		tx("cbc:PayableAmount", amt(lineTotal+taxTotal), "currencyID", cur)))

	for i, l := range d.Lines {
		code := categoryCode(l.Category)
		cat := el("cac:ClassifiedTaxCategory", tx("cbc:ID", code))
		if code == "S" || code == "Z" {
			cat.kids = append(cat.kids, tx("cbc:Percent", pct(l.Rate)))
		}
		if code == "E" {
			cat.kids = append(cat.kids, tx("cbc:TaxExemptionReasonCode", d.ExemptionCode))
		}
		cat.kids = append(cat.kids, el("cac:TaxScheme", tx("cbc:ID", "VAT")))
		item := el("cac:Item", tx("cbc:Description", l.Description), tx("cbc:Name", truncate(l.Description, 100)))
		if d.ServiceSAC != "" {
			item.kids = append(item.kids, el("cac:AdditionalItemIdentification", tx("cbc:ID", d.ServiceSAC, "schemeID", "SAC")))
			item.kids = append(item.kids, el("cac:CommodityClassification", tx("cbc:CommodityCode", "S")))
		}
		item.kids = append(item.kids, cat)
		line := el(lineName,
			tx("cbc:ID", fmt.Sprint(i+1)),
			tx(qtyName, qty(l.Quantity), "unitCode", "C62"),
			tx("cbc:LineExtensionAmount", amt(l.Net), "currencyID", cur),
			item,
			el("cac:Price",
				tx("cbc:PriceAmount", amt(l.UnitPrice), "currencyID", cur),
				tx("cbc:BaseQuantity", "1", "unitCode", "C62"),
				el("cac:AllowanceCharge",
					tx("cbc:ChargeIndicator", "false"),
					tx("cbc:Amount", "0.00", "currencyID", cur),
					tx("cbc:BaseAmount", amt(l.UnitPrice), "currencyID", cur))),
		)
		// Line amount payable (BTAE-10) and line VAT (BTAE-08); exempt lines carry no VAT amount.
		ext := el("cac:ItemPriceExtension", tx("cbc:Amount", amt(l.Net+l.VAT), "currencyID", cur))
		if code != "E" {
			ext.kids = append(ext.kids, el("cac:TaxTotal", tx("cbc:TaxAmount", amt(l.VAT), "currencyID", cur)))
		}
		line.kids = append(line.kids, ext)
		add(line)
	}

	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	doc.write(&b, 0)
	return []byte(b.String()), nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

// ErrNotEnabled is returned when e-invoicing is off for the company.
var ErrNotEnabled = errors.New("e-invoicing is not enabled for this company")
