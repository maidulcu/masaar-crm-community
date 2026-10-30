package handler

import (
	"io"
	"strings"
	"testing"
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
