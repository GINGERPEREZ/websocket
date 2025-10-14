package infrastructure

import (
	"context"
	"encoding/json"
	"log"
	"mesaYaWs/internal/realtime/domain"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	topics map[string]map[*Client]struct{}
	mu     sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{topics: make(map[string]map[*Client]struct{})}
}

func (h *Hub) Register(topic string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[topic] == nil {
		h.topics[topic] = make(map[*Client]struct{})
	}
	h.topics[topic][c] = struct{}{}
}

func (h *Hub) Broadcast(ctx context.Context, msg *domain.Message) {
	h.mu.RLock()
	clients := h.topics[msg.Topic]
	h.mu.RUnlock()

	data, _ := json.Marshal(msg)
	for c := range clients {
		select {
		case c.send <- data:
		default:
			close(c.send)
			c.conn.Close()
			delete(clients, c)
		}
	}
}

func (c *Client) WritePump() {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}
