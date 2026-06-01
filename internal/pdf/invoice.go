package pdf

import (
	"errors"
	"time"
)

var errProOnly = errors.New("PDF generation requires a Pro plan — visit https://masaar.io/pricing")

type InvoiceData struct {
	InvoiceNo   string
	IssuedAt    time.Time
	Subtotal    float64
	VATRate     float64
	VATAmount   float64
	Total       float64
	DealTitle   string
	CompanyName string
	CompanyAddr string
	CompanyVAT  string
}

func GenerateInvoice(data InvoiceData) ([]byte, error) {
	return nil, errProOnly
}
