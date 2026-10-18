// Package totp implements time-based one-time passwords (RFC 6238, HMAC-SHA1, 6 digits,
// 30-second steps) as used by authenticator apps, single-use backup codes, and sealing of
// the shared secrets at rest.
package totp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // RFC 6238 default; what authenticator apps implement
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	digits = 6
	period = 30
	// Skew is how many steps before or after the current one are accepted, for clock drift.
	Skew = 1
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewSecret returns a random 160-bit secret, base32-encoded.
func NewSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return b32.EncodeToString(b), nil
}

// Step returns the time step of t.
func Step(t time.Time) int64 { return t.Unix() / period }

// CodeAt returns the code for a time step.
func CodeAt(secret string, step int64) (string, error) {
	key, err := b32.DecodeString(strings.ToUpper(strings.TrimRight(secret, "=")))
	if err != nil {
		return "", fmt.Errorf("totp: bad secret: %w", err)
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	v := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", digits, v%1_000_000), nil
}

// Verify checks a code at time t, allowing Skew steps of drift. A step at or before
// lastStep is refused, so a code cannot be used twice. It returns the matched step.
func Verify(secret, code string, t time.Time, lastStep int64) (int64, bool) {
	code = strings.TrimSpace(strings.ReplaceAll(code, " ", ""))
	if len(code) != digits {
		return 0, false
	}
	now := Step(t)
	for s := now - Skew; s <= now+Skew; s++ {
		if s <= lastStep {
			continue
		}
		want, err := CodeAt(secret, s)
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return s, true
		}
	}
	return 0, false
}

// URI returns the otpauth:// URI an authenticator app reads from a QR code.
func URI(issuer, account, secret string) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(digits))
	q.Set("period", fmt.Sprint(period))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// ── Backup codes ──────────────────────────────────────────────────────────────

// BackupCodeCount is how many backup codes a user gets.
const BackupCodeCount = 8

const backupAlphabet = "abcdefghjkmnpqrstuvwxyz23456789" // no 0/o, 1/l/i

// NewBackupCodes returns n random codes like "k7m2-x9pq-r4tz" (about 59 bits each).
func NewBackupCodes(n int) ([]string, error) {
	out := make([]string, n)
	buf := make([]byte, 12)
	for i := range out {
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		var b strings.Builder
		for j, c := range buf {
			if j > 0 && j%4 == 0 {
				b.WriteByte('-')
			}
			b.WriteByte(backupAlphabet[int(c)%len(backupAlphabet)])
		}
		out[i] = b.String()
	}
	return out, nil
}

// LooksLikeBackupCode reports whether s has the shape of a backup code rather than a
// six-digit code.
func LooksLikeBackupCode(s string) bool {
	return len(normalizeBackup(s)) == 12
}

func normalizeBackup(s string) string {
	s = strings.ToLower(s)
	return strings.Map(func(r rune) rune {
		if r == '-' || r == ' ' {
			return -1
		}
		return r
	}, s)
}

// HashBackupCode returns the stored form of a backup code. The codes are random and long,
// so a plain SHA-256 is enough.
func HashBackupCode(code string) string {
	sum := sha256.Sum256([]byte("masaar-backup:" + normalizeBackup(code)))
	return hex.EncodeToString(sum[:])
}

// ── Sealing secrets at rest ───────────────────────────────────────────────────

// Box encrypts secrets with AES-256-GCM.
type Box struct{ aead cipher.AEAD }

// NewBox derives a key from keyMaterial (any length).
func NewBox(keyMaterial string) (*Box, error) {
	if keyMaterial == "" {
		return nil, errors.New("totp: empty key")
	}
	k := sha256.Sum256([]byte("masaar-totp-key:" + keyMaterial))
	block, err := aes.NewCipher(k[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

// Seal encrypts a secret.
func (b *Box) Seal(plain string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return "v1:" + base64.RawStdEncoding.EncodeToString(b.aead.Seal(nonce, nonce, []byte(plain), nil)), nil
}

// Open decrypts a sealed secret.
func (b *Box) Open(sealed string) (string, error) {
	raw, ok := strings.CutPrefix(sealed, "v1:")
	if !ok {
		return "", errors.New("totp: unknown secret format")
	}
	data, err := base64.RawStdEncoding.DecodeString(raw)
	if err != nil || len(data) < b.aead.NonceSize() {
		return "", errors.New("totp: corrupt secret")
	}
	plain, err := b.aead.Open(nil, data[:b.aead.NonceSize()], data[b.aead.NonceSize():], nil)
	if err != nil {
		return "", errors.New("totp: cannot decrypt secret (was TWO_FACTOR_KEY changed?)")
	}
	return string(plain), nil
}
