package middleware

import (
	"errors"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/redis/go-redis/v9"
)

func TestRedisLimiterStorageFailsOpen(t *testing.T) {
	db, mock := redismock.NewClientMock()
	s := NewRedisLimiterStorage(db, "login")

	mock.ExpectGet("ratelimit:login:1.2.3.4").SetErr(errors.New("connection refused"))
	if v, err := s.Get("1.2.3.4"); v != nil || err != nil {
		t.Fatalf("Get during outage = %v, %v; want nil, nil", v, err)
	}
	mock.ExpectSet("ratelimit:login:1.2.3.4", []byte("x"), time.Minute).SetErr(errors.New("down"))
	if err := s.Set("1.2.3.4", []byte("x"), time.Minute); err != nil {
		t.Fatalf("Set during outage returned %v", err)
	}
}

func TestRedisLimiterStorageNamespacesKeys(t *testing.T) {
	db, mock := redismock.NewClientMock()
	mock.ExpectSet("ratelimit:sms:ip", []byte("v"), time.Minute).SetVal("OK")
	mock.ExpectGet("ratelimit:sms:ip").SetVal("v")
	s := NewRedisLimiterStorage(db, "sms")
	if err := s.Set("ip", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Get("ip"); string(v) != "v" {
		t.Fatalf("got %q", v)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// With a real Redis (TEST_REDIS_ADDR) two independent apps — standing in for two
// replicas — must share one counter.
func TestLimiterSharedAcrossInstances(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	storage := NewRedisLimiterStorage(rdb, "shared-test")
	_ = storage.Reset()
	t.Cleanup(func() { _ = storage.Reset() })

	build := func() *fiber.App {
		app := fiber.New()
		app.Use(limiter.New(limiter.Config{Max: 3, Expiration: time.Minute, Storage: storage}))
		app.Get("/", func(c *fiber.Ctx) error { return c.SendString("ok") })
		return app
	}
	a, b := build(), build()
	codes := []int{}
	for i, app := range []*fiber.App{a, b, a, b} {
		resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
		if err != nil {
			t.Fatalf("req %d: %v", i, err)
		}
		codes = append(codes, resp.StatusCode)
	}
	if codes[0] != 200 || codes[1] != 200 || codes[2] != 200 || codes[3] != 429 {
		t.Fatalf("status codes = %v; want [200 200 200 429]", codes)
	}
}
