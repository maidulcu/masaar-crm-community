package ws

import (
	"encoding/json"
	"log"
	"sync"

	fiberws "github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
)

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type client struct {
	conn      *fiberws.Conn
	send      chan []byte
	userID    string
	companyID string
}

type Hub struct {
	mu          sync.RWMutex
	clients     map[*client]struct{}
	userClients map[string]map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*client]struct{}),
		userClients: make(map[string]map[*client]struct{}),
	}
}

// BroadcastToCompany delivers an event to every connection that belongs to the
// given company. Events must never cross company boundaries, so there is
// deliberately no unscoped broadcast.
func (h *Hub) BroadcastToCompany(companyID string, e Event) {
	if companyID == "" {
		return
	}
	data, err := json.Marshal(e)
	if err != nil {
		log.Printf("ws hub: marshal: %v", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		if c.companyID != companyID {
			continue
		}
		select {
		case c.send <- data:
		default:
		}
	}
}

func (h *Hub) SendToUser(userID string, e Event) {
	data, err := json.Marshal(e)
	if err != nil {
		log.Printf("ws hub: marshal: %v", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.userClients[userID]; ok {
		for c := range clients {
			select {
			case c.send <- data:
			default:
			}
		}
	}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	if c.userID != "" {
		if h.userClients[c.userID] == nil {
			h.userClients[c.userID] = make(map[*client]struct{})
		}
		h.userClients[c.userID][c] = struct{}{}
	}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	if c.userID != "" {
		if h.userClients[c.userID] != nil {
			delete(h.userClients[c.userID], c)
		}
	}
}

// Handler serves an upgraded WebSocket connection. The user and company are
// taken from the verified JWT (set in Fiber locals by the auth middleware
// before the upgrade) — never from client-supplied query parameters.
func (h *Hub) Handler() func(*fiberws.Conn) {
	return func(conn *fiberws.Conn) {
		userID, ok := conn.Locals("user_id").(uuid.UUID)
		companyID, _ := conn.Locals("company_id").(string)
		if !ok || companyID == "" {
			conn.Close()
			return
		}

		c := &client{
			conn:      conn,
			send:      make(chan []byte, 64),
			userID:    userID.String(),
			companyID: companyID,
		}

		h.register(c)
		defer func() {
			h.unregister(c)
			conn.Close()
		}()

		// done signals the write loop to stop. We never close c.send directly
		// because sending to a closed channel panics even inside a select.
		done := make(chan struct{})

		go func() {
			defer close(done)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		for {
			select {
			case msg, ok := <-c.send:
				if !ok {
					return
				}
				if err := conn.WriteMessage(fiberws.TextMessage, msg); err != nil {
					return
				}
			case <-done:
				return
			}
		}
	}
}
