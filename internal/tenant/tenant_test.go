package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type localsCtx struct {
	context.Context
	vals map[string]any
}

func (l localsCtx) Value(k any) any {
	if s, ok := k.(string); ok {
		return l.vals[s]
	}
	return l.Context.Value(k)
}

func TestFrom(t *testing.T) {
	id := uuid.New()
	other := uuid.New()

	if _, err := From(context.Background()); err != ErrMissing {
		t.Fatalf("empty context must fail closed, got %v", err)
	}
	if got, err := From(With(context.Background(), id)); err != nil || got != id {
		t.Fatalf("With: got %v %v", got, err)
	}
	loc := localsCtx{context.Background(), map[string]any{"company_id": other.String()}}
	if got, err := From(loc); err != nil || got != other {
		t.Fatalf("locals: got %v %v", got, err)
	}
	// explicit binding wins over request locals
	if got, _ := From(With(loc, id)); got != id {
		t.Fatalf("With must take precedence, got %v", got)
	}
	for _, bad := range []any{"", "not-a-uuid", uuid.Nil.String(), 42} {
		if _, err := From(localsCtx{context.Background(), map[string]any{"company_id": bad}}); err != ErrMissing {
			t.Fatalf("invalid locals %v must fail closed, got %v", bad, err)
		}
	}
	if _, err := From(With(context.Background(), uuid.Nil)); err != ErrMissing {
		t.Fatal("nil UUID must fail closed")
	}
}
