package handler

import "testing"

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
		"  =1+1":                     "'  =1+1",
		"\n=1+1":                     "'\n=1+1",
		"|cmd|' /C calc'!A0":         "'|cmd|' /C calc'!A0",
		"%0A=1+1":                    "'%0A=1+1",
		"  +971 50 123 4567":         "'  +971 50 123 4567",
	}
	for in, want := range tests {
		if got := csvSafeCell(in); got != want {
			t.Errorf("csvSafeCell(%q) = %q, want %q", in, got, want)
		}
	}
}
