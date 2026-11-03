package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

// Idempotency-Key support for API-key writes, following the IETF httpapi
// "The Idempotency-Key HTTP Header Field" draft: a retried POST or PATCH with the same key and
// the same request gets the first response back instead of acting twice.

const (
	IdempotencyTTL = 24 * time.Hour
	// idempotencyLockTTL bounds how long an unfinished request holds its key, so a crash
	// mid-request does not block retries for a day.
	idempotencyLockTTL = 2 * time.Minute
)

var (
	ErrIdempotencyInFlight = errors.New("idempotency: request in progress")
	ErrIdempotencyMismatch = errors.New("idempotency: key reused with a different request")
)

// IdempotentResponse is what a store keeps per key: the request's fingerprint and, once Done,
// the response to replay.
type IdempotentResponse struct {
	Fingerprint string `json:"fp"`
	Done        bool   `json:"done"`
	Status      int    `json:"status,omitempty"`
	ContentType string `json:"ctype,omitempty"`
	Location    string `json:"location,omitempty"`
	Body        []byte `json:"body,omitempty"`
}

// IdempotencyStore holds keys. Reserve claims a new key and returns (nil, nil), or returns the
// finished response of an earlier request, ErrIdempotencyInFlight or ErrIdempotencyMismatch.
type IdempotencyStore interface {
	Reserve(ctx context.Context, key, fingerprint string, ttl time.Duration) (*IdempotentResponse, error)
	Complete(ctx context.Context, key string, resp IdempotentResponse, ttl time.Duration) error
	Release(ctx context.Context, key string) error
}

// Idempotency applies to POST and PATCH requests made with an API key that send an
// Idempotency-Key header. Keys are scoped to the API key. If the store fails the request runs
// without the guarantee rather than failing.
func Idempotency(store IdempotencyStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Get("Idempotency-Key")
		m := c.Method()
		owner := idempotencyOwner(c)
		if key == "" || store == nil || owner == "" || (m != fiber.MethodPost && m != fiber.MethodPatch) {
			return c.Next()
		}
		if !validIdempotencyKey(key) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Idempotency-Key must be 1-255 printable ASCII characters"})
		}
		scoped := fmt.Sprintf("idem:%s:%s", owner, key)
		sum := sha256.Sum256([]byte(m + " " + c.Path() + "?" + string(c.Request().URI().QueryString()) + "\n" + string(c.Body())))
		fp := hex.EncodeToString(sum[:])

		saved, err := store.Reserve(c.Context(), scoped, fp, idempotencyLockTTL)
		switch {
		case errors.Is(err, ErrIdempotencyMismatch):
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "Idempotency-Key was already used with a different request"})
		case errors.Is(err, ErrIdempotencyInFlight):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "a request with this Idempotency-Key is still being processed"})
		case err != nil:
			return c.Next()
		case saved != nil:
			c.Set("Idempotent-Replayed", "true")
			if saved.ContentType != "" {
				c.Set(fiber.HeaderContentType, saved.ContentType)
			}
			if saved.Location != "" {
				c.Set(fiber.HeaderLocation, saved.Location)
			}
			return c.Status(saved.Status).Send(saved.Body)
		}

		// A failure (error, 5xx, panic) may succeed on retry: free the key unless it completed.
		completed := false
		defer func() {
			if !completed {
				_ = store.Release(context.Background(), scoped)
			}
		}()
		herr := c.Next()
		status := c.Response().StatusCode()
		if herr != nil || status >= 500 {
			return herr
		}
		// Refused for authentication or permission (e.g. a missing scope): not the request's
		// outcome. Once the client has the access, a retry with the same key must run.
		if status == fiber.StatusUnauthorized || status == fiber.StatusForbidden {
			return nil
		}
		// A response carrying a secret (a new key, a login link) is marked no-store and is not
		// kept for replay; the key is released and a retry runs again.
		if strings.Contains(string(c.Response().Header.Peek(fiber.HeaderCacheControl)), "no-store") {
			return nil
		}
		if store.Complete(c.Context(), scoped, IdempotentResponse{
			Fingerprint: fp,
			Done:        true,
			Status:      status,
			ContentType: string(c.Response().Header.ContentType()),
			Location:    string(c.Response().Header.Peek(fiber.HeaderLocation)),
			Body:        append([]byte(nil), c.Response().Body()...),
		}, IdempotencyTTL) == nil {
			completed = true
		}
		return nil
	}
}

// idempotencyOwner is who the key belongs to: the API key, or a partner client. Session requests
// get no idempotency handling.
func idempotencyOwner(c *fiber.Ctx) string {
	if id := c.Locals("api_key_id"); id != nil {
		return fmt.Sprint(id)
	}
	if p, ok := c.Locals("partner_client_id").(string); ok && p != "" {
		return "partner:" + p
	}
	return ""
}

func validIdempotencyKey(k string) bool {
	if len(k) == 0 || len(k) > 255 {
		return false
	}
	for i := 0; i < len(k); i++ {
		if k[i] < 0x20 || k[i] > 0x7e {
			return false
		}
	}
	return true
}

// RedisIdempotencyStore keeps keys in Redis with a TTL.
type RedisIdempotencyStore struct{ rdb *redis.Client }

// NewRedisIdempotencyStore returns nil when rdb is nil, which disables Idempotency.
func NewRedisIdempotencyStore(rdb *redis.Client) IdempotencyStore {
	if rdb == nil {
		return nil
	}
	return &RedisIdempotencyStore{rdb: rdb}
}

func (s *RedisIdempotencyStore) Reserve(ctx context.Context, key, fp string, ttl time.Duration) (*IdempotentResponse, error) {
	pending, _ := json.Marshal(IdempotentResponse{Fingerprint: fp})
	ok, err := s.rdb.SetNX(ctx, key, pending, ttl).Result()
	if err != nil {
		return nil, err
	}
	if ok {
		return nil, nil
	}
	raw, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	return decideIdempotent(raw, fp)
}

func (s *RedisIdempotencyStore) Complete(ctx context.Context, key string, resp IdempotentResponse, ttl time.Duration) error {
	raw, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return s.rdb.Set(ctx, key, raw, ttl).Err()
}

func (s *RedisIdempotencyStore) Release(ctx context.Context, key string) error {
	return s.rdb.Del(ctx, key).Err()
}

// decideIdempotent judges a stored record against a new request's fingerprint.
func decideIdempotent(raw []byte, fp string) (*IdempotentResponse, error) {
	var rec IdempotentResponse
	if err := json.Unmarshal(raw, &rec); err != nil {
		return nil, err
	}
	if rec.Fingerprint != fp {
		return nil, ErrIdempotencyMismatch
	}
	if !rec.Done {
		return nil, ErrIdempotencyInFlight
	}
	return &rec, nil
}
