package infrastructure

import (
	"context"
	"encoding/json"
	"log/slog"
	"mesaYaWs/internal/modules/realtime/domain"
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
	sectionID  string
	entity     string
	token      string
	commandFn  func(context.Context, *Client, Command)
	subscribed map[string]struct{}
	closeOnce  sync.Once
}

type Command struct {
	Action  string          `json:"action"`
	Topic   string          `json:"topic,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// NewClient crea un cliente WebSocket con metadata de usuario y buffer configurable.
func NewClient(hub *Hub, conn *websocket.Conn, userID, sessionID, sectionID, entity, token string, buf int, commandFn func(context.Context, *Client, Command)) *Client {
	return &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, buf),
		userID:     userID,
		sessionID:  sessionID,
		sectionID:  strings.TrimSpace(sectionID),
		entity:     strings.TrimSpace(entity),
		token:      token,
		commandFn:  commandFn,
		subscribed: make(map[string]struct{}),
	}
}

func (c *Client) key() string {
	parts := []string{c.userID, c.sessionID}
	if c.sectionID != "" {
		parts = append(parts, c.sectionID)
	}
	return strings.Join(parts, ":")
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		close(c.send)
		_ = c.conn.Close()
	})
}

func (c *Client) SendDomainMessage(msg *domain.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("websocket marshal error", slog.Any("error", err))
		return
	}
	select {
	case c.send <- data:
	default:
		slog.Warn("websocket send buffer full", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID))
		go c.hub.detachClient(c)
	}
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
				slog.Warn("websocket write error", slog.Any("error", err))
				return
			}
		case <-ping.C:
			if err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second)); err != nil {
				slog.Warn("websocket ping error", slog.Any("error", err))
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
		var cmd Command
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		if err := c.conn.ReadJSON(&cmd); err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				slog.Warn("websocket read error", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID), slog.Any("error", err))
			}
			return
		}
		c.handleCommand(cmd)
	}
}

func (c *Client) handleCommand(cmd Command) {
	switch strings.ToLower(cmd.Action) {
	case "subscribe":
		if cmd.Topic != "" {
			c.hub.subscribe(c, cmd.Topic)
			slog.Debug("ws subscribe", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID), slog.String("topic", cmd.Topic))
		}
	case "unsubscribe":
		if cmd.Topic != "" {
			c.hub.unsubscribe(c, cmd.Topic)
			slog.Debug("ws unsubscribe", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID), slog.String("topic", cmd.Topic))
		}
	case "ping":
		ack := domain.Message{
			Topic:     "system.pong",
			Entity:    "system",
			Action:    "pong",
			Timestamp: time.Now().UTC(),
		}
		c.SendDomainMessage(&ack)
	default:
		if c.commandFn != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			go func() {
				defer cancel()
				c.commandFn(ctx, c, cmd)
			}()
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
	slog.Info("ws client registered", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID))
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
	slog.Debug("ws client unsubscribed", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID), slog.String("topic", topic))
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
	slog.Info("ws client detached", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID))
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
		slog.Error("broadcast marshal error", slog.Any("error", err))
		return
	}

	targetUser := ""
	targetSession := ""
	targetSection := ""
	if msg.Metadata != nil {
		targetUser = strings.TrimSpace(msg.Metadata["userId"])
		targetSession = strings.TrimSpace(msg.Metadata["sessionId"])
		targetSection = strings.TrimSpace(msg.Metadata["sectionId"])
	}

	for _, c := range clients {
		if targetUser != "" && c.userID != targetUser {
			continue
		}
		if targetSession != "" && c.sessionID != targetSession {
			continue
		}
		if targetSection != "" && c.sectionID != targetSection {
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
	slog.Info("ws client attached", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID), slog.Any("topics", topics))
}
