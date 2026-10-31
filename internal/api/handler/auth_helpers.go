package handler

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"github.com/maidulcu/masaar-crm/internal/domain"
)

const (
	maxNameRunes    = 100
	maxCompanyRunes = 150
	maxEmailLen     = 254
	// bcrypt only uses the first 72 bytes and newer versions refuse longer input outright, which
	// used to surface as a 500 "failed to hash password".
	maxPasswordBytes = 72

	// Five wrong passwords lock an email for the address (IP) they came from. Locking the email
	// everywhere after five let anyone lock a user out by guessing badly on purpose; the much
	// higher per-email cap still stops a guessing run spread over many addresses.
	loginMaxFailures         = 5
	loginMaxFailuresPerEmail = 50
	loginLockout             = 15 * time.Minute
)

var subdomainRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,38}[a-z0-9]$`)

// reservedSubdomains cannot be claimed by a workspace: they are, or may become, real hostnames.
var reservedSubdomains = map[string]bool{
	"www": true, "api": true, "app": true, "admin": true, "mail": true, "smtp": true, "ftp": true,
	"localhost": true, "static": true, "assets": true, "cdn": true, "docs": true, "support": true,
	"status": true, "login": true, "signup": true, "auth": true, "demo": true, "dashboard": true,
}

// registrationFields is a sign-up request after trimming and normalisation.
type registrationFields struct {
	Name, Email, CompanyName, Subdomain, Password string
}

// normalizeRegistration trims and validates sign-up input. The returned message is empty when
// the input is acceptable. Emails are lower-cased here because login looks them up that way: an
// account stored with capitals could never log in.
func normalizeRegistration(name, email, password, companyName, subdomain string) (registrationFields, string) {
	f := registrationFields{
		Name:        strings.TrimSpace(name),
		Email:       strings.ToLower(strings.TrimSpace(email)),
		Password:    password,
		CompanyName: strings.TrimSpace(companyName),
		Subdomain:   strings.ToLower(strings.TrimSpace(subdomain)),
	}
	if f.Name == "" || f.Email == "" || f.Password == "" {
		return f, "name, email, and password are required"
	}
	if f.CompanyName == "" || f.Subdomain == "" {
		return f, "company_name and subdomain are required"
	}
	if utf8.RuneCountInString(f.Name) > maxNameRunes {
		return f, "name is too long"
	}
	if utf8.RuneCountInString(f.CompanyName) > maxCompanyRunes {
		return f, "company_name is too long"
	}
	if addr, err := mail.ParseAddress(f.Email); err != nil || addr.Address != f.Email || len(f.Email) > maxEmailLen {
		return f, "invalid email format"
	}
	if !subdomainRE.MatchString(f.Subdomain) {
		return f, "subdomain must be 3-40 characters: lowercase letters, digits and hyphens, not starting or ending with a hyphen"
	}
	if reservedSubdomains[f.Subdomain] {
		return f, "subdomain_taken"
	}
	if msg := validatePasswordStrength(f.Password); msg != "" {
		return f, msg
	}
	return f, ""
}

// allowRate counts one hit against key and reports whether it is within max per window. The
// increment is atomic (unlike read-then-increment, concurrent requests cannot all slip under the
// limit) and the expiry is (re)armed whenever the key has none, so a crash between the two calls
// can never leave a counter that never resets.
func allowRate(ctx context.Context, rdb *redis.Client, key string, max int, window time.Duration) (bool, error) {
	n, err := rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		if err := rdb.Expire(ctx, key, window).Err(); err != nil {
			return false, err
		}
	} else if ttl, err := rdb.TTL(ctx, key).Result(); err == nil && ttl < 0 {
		_ = rdb.Expire(ctx, key, window).Err()
	}
	return n <= int64(max), nil
}

// loginKeys are the Redis keys of an email's sign-in failures: per source address, and for the
// email as a whole. "lockout:<email>" is also set by too many wrong two-factor codes.
func loginKeys(email, ip string) (failIP, lockIP, failAll, lockAll string) {
	return fmt.Sprintf("loginfail:%s|%s", email, ip), fmt.Sprintf("lockout:%s|%s", email, ip),
		fmt.Sprintf("loginfail:%s", email), fmt.Sprintf("lockout:%s", email)
}

// loginLocked reports whether sign-in for email is locked, for this address or everywhere.
func (h *AuthHandler) loginLocked(ctx context.Context, email, ip string) bool {
	_, lockIP, _, lockAll := loginKeys(email, ip)
	n, _ := h.redis.Exists(ctx, lockIP, lockAll).Result()
	return n > 0
}

// recordLoginFailure counts a failed sign-in for email from ip and reports whether it is now
// locked. It is applied identically to known and unknown emails: otherwise the "account locked"
// response would only ever appear for registered addresses and reveal which ones exist.
func (h *AuthHandler) recordLoginFailure(ctx context.Context, email, ip string) (locked bool) {
	failIP, lockIP, failAll, lockAll := loginKeys(email, ip)
	if n := h.countFailure(ctx, failAll); n >= loginMaxFailuresPerEmail {
		h.redis.Set(ctx, lockAll, "1", loginLockout)
		h.redis.Del(ctx, failAll)
		locked = true
	}
	if n := h.countFailure(ctx, failIP); n >= loginMaxFailures {
		h.redis.Set(ctx, lockIP, "1", loginLockout)
		h.redis.Del(ctx, failIP)
		locked = true
	}
	return locked
}

func (h *AuthHandler) countFailure(ctx context.Context, key string) int64 {
	n, _ := h.redis.Incr(ctx, key).Result()
	h.redis.Expire(ctx, key, loginLockout)
	return n
}

// clearLoginFailures resets the lockout state for email: everywhere, and for ip (after a
// successful login or password reset from that address).
func (h *AuthHandler) clearLoginFailures(ctx context.Context, email, ip string) {
	failIP, lockIP, failAll, lockAll := loginKeys(email, ip)
	h.redis.Del(ctx, failIP, lockIP, failAll, lockAll)
}

var errAccountSuspended = errors.New("account suspended")

// loadCompany loads the user's company for a sign-in/refresh, refusing suspended companies and
// downgrading an expired trial. It returns the whole days of trial left. With no company
// repository configured it returns (nil, 0, nil).
func (h *AuthHandler) loadCompany(ctx context.Context, user *domain.User) (*domain.Company, int, error) {
	if h.companyRepo == nil {
		return nil, 0, nil
	}
	company, err := h.companyRepo.GetByID(ctx, user.CompanyID)
	if err != nil {
		return nil, 0, err
	}
	if !company.IsActive {
		return nil, 0, errAccountSuspended
	}
	days := 0
	if company.OnTrial && company.TrialEndsAt != nil {
		if company.TrialEndsAt.Before(time.Now()) {
			_ = h.companyRepo.EndTrial(ctx, user.CompanyID)
			company.Plan = "community"
			company.OnTrial = false
		} else {
			days = int(time.Until(*company.TrialEndsAt).Hours() / 24)
		}
	}
	return company, days, nil
}

// companyPayload is the company object returned by every sign-in endpoint.
func companyPayload(c *domain.Company, daysRemaining int) fiber.Map {
	return fiber.Map{
		"id":             c.ID,
		"name":           c.Name,
		"subdomain":      c.Subdomain,
		"plan":           c.Plan,
		"on_trial":       c.OnTrial,
		"trial_ends_at":  c.TrialEndsAt,
		"days_remaining": daysRemaining,
		"is_demo":        c.IsDemo,
	}
}

// companyError maps a loadCompany failure to its HTTP response.
func companyError(c *fiber.Ctx, err error) error {
	if errors.Is(err, errAccountSuspended) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "account_suspended"})
	}
	return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "company not found"})
}
