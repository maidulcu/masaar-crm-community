package totp

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

// RFC 6238 appendix B, SHA-1 seed "12345678901234567890"; the RFC lists 8 digits, these
// are their last 6.
func TestRFC6238Vectors(t *testing.T) {
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte("12345678901234567890"))
	for unix, want := range map[int64]string{59: "287082", 1111111109: "081804", 1111111111: "050471", 1234567890: "005924", 2000000000: "279037"} {
		got, err := CodeAt(secret, Step(time.Unix(unix, 0)))
		if err != nil || got != want {
			t.Errorf("t=%d: %s (%v), want %s", unix, got, err, want)
		}
	}
}

func TestVerifyWindowAndReplay(t *testing.T) {
	secret, _ := NewSecret()
	now := time.Unix(1_800_000_000, 0)
	prev, _ := CodeAt(secret, Step(now)-1)
	step, ok := Verify(secret, prev, now, 0)
	if !ok || step != Step(now)-1 {
		t.Fatal("a code from the previous step is refused")
	}
	if _, ok := Verify(secret, prev, now, step); ok {
		t.Fatal("a code was accepted twice")
	}
	old, _ := CodeAt(secret, Step(now)-3)
	if _, ok := Verify(secret, old, now, 0); ok {
		t.Fatal("a code 90 s old was accepted")
	}
	if _, ok := Verify(secret, "12345", now, 0); ok {
		t.Fatal("a 5-digit code was accepted")
	}
	if !strings.HasPrefix(URI("Masaar CRM", "a@b.test", secret), "otpauth://totp/Masaar%20CRM:a@b.test?") {
		t.Error(URI("Masaar CRM", "a@b.test", secret))
	}
}

func TestBackupCodes(t *testing.T) {
	codes, err := NewBackupCodes(BackupCodeCount)
	if err != nil || len(codes) != 8 {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if len(c) != 14 || !LooksLikeBackupCode(c) || seen[c] {
			t.Fatalf("bad code %q", c)
		}
		seen[c] = true
	}
	if HashBackupCode(codes[0]) != HashBackupCode(" "+strings.ToUpper(strings.ReplaceAll(codes[0], "-", ""))+" ") {
		t.Error("hash is not normalised")
	}
	if LooksLikeBackupCode("123456") {
		t.Error("a TOTP code looks like a backup code")
	}
}

func TestBox(t *testing.T) {
	b, _ := NewBox("key one")
	sealed, err := b.Seal("JBSWY3DPEHPK3PXP")
	if err != nil || strings.Contains(sealed, "JBSWY3DP") {
		t.Fatal(sealed, err)
	}
	if got, err := b.Open(sealed); err != nil || got != "JBSWY3DPEHPK3PXP" {
		t.Fatal(got, err)
	}
	other, _ := NewBox("key two")
	if _, err := other.Open(sealed); err == nil {
		t.Fatal("opened with the wrong key")
	}
}
