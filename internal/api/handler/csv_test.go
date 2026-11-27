package handler

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestCSVSafeCell(t *testing.T) {
	tests := map[string]string{
		"":                           "",
		"Ahmed":                      "Ahmed",
		"+971501234567":              "+971501234567",
		"+971 50 123 4567":           "+971 50 123 4567",
		"-5.00":                      "-5.00",
		"1500.50":                    "1500.50",
		`=HYPERLINK("http://e","x")`: `'=HYPERLINK("http://e","x")`,
		"=1+1":                       "'=1+1",
		"+cmd|' /C calc'!A0":         "'+cmd|' /C calc'!A0",
		"-2+3":                       "'-2+3", // an actual formula
		"@SUM(A1)":                   "'@SUM(A1)",
		"\t=1":                       "'\t=1",
		"hello=world":                "hello=world",
		" =1+1":                      "' =1+1",
		"\n=1+1":                     "'\n=1+1",
		"%SUM(A1)":                   "'%SUM(A1)",
	}
	for in, want := range tests {
		if got := csvSafeCell(in); got != want {
			t.Errorf("csvSafeCell(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMaxImportFileSizeConstant(t *testing.T) {
	if maxImportFileSize != 5<<20 {
		t.Errorf("maxImportFileSize = %d, want %d", maxImportFileSize, 5<<20)
	}
}

func TestParseCSVLimitedReaderOversized(t *testing.T) {
	// Create an oversized stream that exceeds maxImportFileSize
	oversized := strings.Repeat("full_name,phone_wa\nAhmed,+971501234567\n", 300000)
	r := strings.NewReader(oversized)
	if int64(r.Len()) <= maxImportFileSize {
		t.Fatalf("test stream length %d must exceed maxImportFileSize %d", r.Len(), maxImportFileSize)
	}

	lr := &io.LimitedReader{R: r, N: maxImportFileSize + 1}
	_, _ = parseCSV(lr)
	if lr.N != 0 {
		t.Errorf("expected lr.N == 0 for oversized stream, got %d", lr.N)
	}
}

func TestCSVExportSecurityHeaders(t *testing.T) {
	h := NewImportExportHandler(nil, nil, nil)
	app := fiber.New()
	app.Get("/template/:entity", h.Template)

	req := httptest.NewRequest("GET", "/template/contacts", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", resp.Header.Get("X-Content-Type-Options"))
	}
	if resp.Header.Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/csv; charset=utf-8", resp.Header.Get("Content-Type"))
	}
}
