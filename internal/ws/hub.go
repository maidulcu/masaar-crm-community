package ws

import (
	"encoding/json"
	"log"
	"sync"

	fiberws "github.com/gofiber/websocket/v2"
)

type Event struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

type client struct {
	conn   *fiberws.Conn
	send   chan []byte
	userID string
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

func (h *Hub) Broadcast(e Event) {
	data, err := json.Marshal(e)
	if err != nil {
		log.Printf("ws hub: marshal: %v", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
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

func (h *Hub) Handler() func(*fiberws.Conn) {
	return func(conn *fiberws.Conn) {
		userID := conn.Query("user")

		c := &client{
			conn:   conn,
			send:   make(chan []byte, 64),
			userID: userID,
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
