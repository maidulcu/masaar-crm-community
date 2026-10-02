package ws

import "testing"

func newTestClient(userID, companyID string) *client {
	return &client{send: make(chan []byte, 4), userID: userID, companyID: companyID}
}

func TestBroadcastToCompany_DoesNotCrossTenants(t *testing.T) {
	h := NewHub()
	a1 := newTestClient("u1", "company-a")
	a2 := newTestClient("u2", "company-a")
	b1 := newTestClient("u3", "company-b")
	for _, c := range []*client{a1, a2, b1} {
		h.register(c)
	}

	h.BroadcastToCompany("company-a", Event{Type: "lead.created", Payload: "secret-a"})

	for name, c := range map[string]*client{"a1": a1, "a2": a2} {
		if len(c.send) != 1 {
			t.Errorf("%s should have received the event, queue=%d", name, len(c.send))
		}
	}
	if len(b1.send) != 0 {
		t.Fatalf("company-b client received company-a event")
	}
}

func TestBroadcastToCompany_EmptyCompanyIsDropped(t *testing.T) {
	h := NewHub()
	orphan := newTestClient("u1", "")
	h.register(orphan)

	h.BroadcastToCompany("", Event{Type: "x"})

	if len(orphan.send) != 0 {
		t.Fatal("an empty company id must never match clients")
	}
}

func TestSendToUser_OnlyTargetReceives(t *testing.T) {
	h := NewHub()
	target := newTestClient("u1", "company-a")
	other := newTestClient("u2", "company-a")
	h.register(target)
	h.register(other)

	h.SendToUser("u1", Event{Type: "notification"})

	if len(target.send) != 1 || len(other.send) != 0 {
		t.Fatalf("target=%d other=%d", len(target.send), len(other.send))
	}
}
