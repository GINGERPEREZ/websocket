package infrastructure

import (
	"context"
	"encoding/json"
	"log"
	"mesaYaWs/internal/realtime/domain"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	userID     string
	sessionID  string
	subscribed map[string]struct{}
	closeOnce  sync.Once
}

type command struct {
	Action string `json:"action"`
	Topic  string `json:"topic"`
}

// NewClient crea un cliente WebSocket con metadata de usuario y buffer configurable.
func NewClient(hub *Hub, conn *websocket.Conn, userID, sessionID string, buf int) *Client {
	return &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, buf),
		userID:     userID,
		sessionID:  sessionID,
		subscribed: make(map[string]struct{}),
	}
}

func (c *Client) key() string {
	return c.userID + ":" + c.sessionID
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.send)
		_ = c.conn.Close()
	})
}

func (c *Client) WritePump() {
	ping := time.NewTicker(30 * time.Second)
	defer ping.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("websocket write error: %v", err)
				return
			}
		case <-ping.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				log.Printf("websocket ping error: %v", err)
				return
			}
		}
	}
}

func (c *Client) ReadPump() {
	c.conn.SetReadLimit(1 << 16)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	defer c.hub.detachClient(c)
	for {
		var cmd command
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if err := c.conn.ReadJSON(&cmd); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("websocket read error: %v", err)
			}
			return
		}
		c.handleCommand(cmd)
	}
}

func (c *Client) handleCommand(cmd command) {
	switch strings.ToLower(cmd.Action) {
	case "subscribe":
		if cmd.Topic != "" {
			c.hub.subscribe(c, cmd.Topic)
		}
	case "unsubscribe":
		if cmd.Topic != "" {
			c.hub.unsubscribe(c, cmd.Topic)
		}
	case "ping":
		ack := domain.Message{
			Topic:     "system.pong",
			Entity:    "system",
			Action:    "pong",
			Timestamp: time.Now().UTC(),
		}
		if data, err := json.Marshal(ack); err == nil {
			select {
			case c.send <- data:
			default:
				log.Printf("websocket: unable to enqueue pong for user %s", c.userID)
			}
		}
	}
}

type Hub struct {
	topics  map[string]map[*Client]struct{}
	clients map[string]*Client
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		topics:  make(map[string]map[*Client]struct{}),
		clients: make(map[string]*Client),
	}
}

func (h *Hub) registerClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if existing, ok := h.clients[c.key()]; ok && existing != c {
		h.detachLocked(existing)
	}
	h.clients[c.key()] = c
}

func (h *Hub) subscribe(c *Client, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.topics[topic] == nil {
		h.topics[topic] = make(map[*Client]struct{})
	}
	h.topics[topic][c] = struct{}{}
	c.subscribed[topic] = struct{}{}
}

func (h *Hub) unsubscribe(c *Client, topic string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs, ok := h.topics[topic]; ok {
		delete(subs, c)
		if len(subs) == 0 {
			delete(h.topics, topic)
		}
	}
	delete(c.subscribed, topic)
}

func (h *Hub) detachClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.detachLocked(c)
}

func (h *Hub) detachLocked(c *Client) {
	if c == nil {
		return
	}
	for topic := range c.subscribed {
		if subs, ok := h.topics[topic]; ok {
			delete(subs, c)
			if len(subs) == 0 {
				delete(h.topics, topic)
			}
		}
	}
	delete(h.clients, c.key())
	c.close()
}

func (h *Hub) Broadcast(_ context.Context, msg *domain.Message) {
	h.mu.RLock()
	clientsMap := h.topics[msg.Topic]
	clients := make([]*Client, 0, len(clientsMap))
	for c := range clientsMap {
		clients = append(clients, c)
	}
	data, err := json.Marshal(msg)
	h.mu.RUnlock()
	if err != nil {
		log.Printf("broadcast marshal error: %v", err)
		return
	}

	targetUser := ""
	targetSession := ""
	if msg.Metadata != nil {
		targetUser = strings.TrimSpace(msg.Metadata["userId"])
		targetSession = strings.TrimSpace(msg.Metadata["sessionId"])
	}

	for _, c := range clients {
		if targetUser != "" && c.userID != targetUser {
			continue
		}
		if targetSession != "" && c.sessionID != targetSession {
			continue
		}
		select {
		case c.send <- data:
		default:
			go h.detachClient(c)
		}
	}
}

func (h *Hub) AttachClient(c *Client, topics []string) {
	h.registerClient(c)
	for _, topic := range topics {
		if strings.TrimSpace(topic) == "" {
			continue
		}
		h.subscribe(c, topic)
	}
}
