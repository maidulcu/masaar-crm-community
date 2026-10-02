package session

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestIssuedBeforeCutoff(t *testing.T) {
	id := uuid.New()
	now := time.Now()
	tests := []struct {
		name   string
		stored string // "" => no key
		iat    time.Time
		want   bool
	}{
		{"no cutoff", "", now.Add(-time.Hour), false},
		{"token older than cutoff", "1000", time.Unix(900, 0), true},
		{"token newer than cutoff", "1000", time.Unix(1100, 0), false},
		{"issued in the same second stays valid", "1000", time.Unix(1000, 0), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rdb, mock := redismock.NewClientMock()
			if tt.stored == "" {
				mock.ExpectGet(key(id)).RedisNil()
			} else {
				mock.ExpectGet(key(id)).SetVal(tt.stored)
			}
			got, err := IssuedBeforeCutoff(context.Background(), rdb, id, tt.iat)
			if err != nil || got != tt.want {
				t.Fatalf("got %v err %v, want %v", got, err, tt.want)
			}
		})
	}
}

func TestRedisFailureIsReportedSoCallersFailClosed(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	id := uuid.New()
	mock.ExpectGet(key(id)).SetErr(errors.New("redis down"))
	if _, err := IssuedBeforeCutoff(context.Background(), rdb, id, time.Now()); err == nil {
		t.Fatal("a Redis error must surface")
	}
	mock.ExpectGet(key(id)).SetVal("not-a-number")
	if blocked, err := IssuedBeforeCutoff(context.Background(), rdb, id, time.Now()); err == nil || !blocked {
		t.Fatalf("a corrupt cutoff must fail closed, got blocked=%v err=%v", blocked, err)
	}
}

func TestRevokeAllWritesCutoff(t *testing.T) {
	rdb, mock := redismock.NewClientMock()
	id := uuid.New()
	mock.Regexp().ExpectSet(key(id), `^\d+$`, cutoffTTL).SetVal("OK")
	if err := RevokeAll(context.Background(), rdb, id); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	var _ *redis.Client = rdb
}
