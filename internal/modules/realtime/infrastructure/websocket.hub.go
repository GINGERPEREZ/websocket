package infrastructure

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"mesaYaWs/internal/modules/realtime/domain"
)

type Command struct {
	Action  string          `json:"action"`
	Topic   string          `json:"topic,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func (c Command) actionKey() string {
	return normalizeAction(c.Action)
}

type CommandHandler func(ctx context.Context, client *Client, cmd Command)

type CommandProcessor struct {
	hub             *Hub
	handlers        map[string]CommandHandler
	fallback        CommandHandler
	fallbackTimeout time.Duration
}

func NewCommandProcessor(hub *Hub, fallback CommandHandler) *CommandProcessor {
	processor := &CommandProcessor{
		hub:             hub,
		handlers:        make(map[string]CommandHandler),
		fallback:        fallback,
		fallbackTimeout: 10 * time.Second,
	}
	processor.Register("subscribe", processor.handleSubscribe)
	processor.Register("unsubscribe", processor.handleUnsubscribe)
	processor.Register("ping", processor.handlePing)
	return processor
}

func (p *CommandProcessor) Register(action string, handler CommandHandler) {
	if handler == nil {
		return
	}
	key := normalizeAction(action)
	if key == "" {
		return
	}
	p.handlers[key] = handler
}

func (p *CommandProcessor) Process(client *Client, cmd Command) {
	if client == nil {
		return
	}

	action := cmd.actionKey()
	if action == "" {
		return
	}

	if handler, ok := p.handlers[action]; ok {
		handler(context.Background(), client, cmd)
		return
	}

	if p.fallback == nil {
		slog.Debug("ws command ignored", slog.String("userId", client.userID), slog.String("sessionId", client.sessionID), slog.String("sectionId", client.sectionID), slog.String("action", action))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), p.fallbackTimeout)
	go func() {
		defer cancel()
		p.fallback(ctx, client, cmd)
	}()
}

func (p *CommandProcessor) handleSubscribe(_ context.Context, client *Client, cmd Command) {
	topic := strings.TrimSpace(cmd.Topic)
	if topic == "" {
		slog.Debug("ws subscribe ignored empty topic", slog.String("userId", client.userID), slog.String("sessionId", client.sessionID), slog.String("sectionId", client.sectionID))
		return
	}
	p.hub.subscribe(client, topic)
	slog.Debug("ws subscribe", slog.String("userId", client.userID), slog.String("sessionId", client.sessionID), slog.String("sectionId", client.sectionID), slog.String("topic", topic))
}

func (p *CommandProcessor) handleUnsubscribe(_ context.Context, client *Client, cmd Command) {
	topic := strings.TrimSpace(cmd.Topic)
	if topic == "" {
		return
	}
	p.hub.unsubscribe(client, topic)
	slog.Debug("ws unsubscribe", slog.String("userId", client.userID), slog.String("sessionId", client.sessionID), slog.String("sectionId", client.sectionID), slog.String("topic", topic))
}

func (p *CommandProcessor) handlePing(_ context.Context, client *Client, _ Command) {
	ack := domain.Message{
		Topic:     "system.pong",
		Entity:    "system",
		Action:    "pong",
		Timestamp: time.Now().UTC(),
	}
	client.SendDomainMessage(&ack)
}

type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	userID     string
	sessionID  string
	sectionID  string
	entity     string
	token      string
	commands   *CommandProcessor
	subscribed map[string]struct{}
	closeOnce  sync.Once
	receiveAll bool
	closeHooks []func(*Client)
	hookMu     sync.Mutex
}

// NewClient crea un cliente WebSocket con metadata de usuario y buffer configurable.
func NewClient(hub *Hub, conn *websocket.Conn, userID, sessionID, sectionID, entity, token string, buf int, commandFn CommandHandler) *Client {
	client := &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, buf),
		userID:     userID,
		sessionID:  sessionID,
		sectionID:  strings.TrimSpace(sectionID),
		entity:     strings.TrimSpace(entity),
		token:      token,
		subscribed: make(map[string]struct{}),
	}
	client.commands = NewCommandProcessor(hub, commandFn)
	return client
}

// EnableReceiveAll marks the client as a global subscriber that receives every broadcasted message
// regardless of topic-specific subscriptions.
func (c *Client) EnableReceiveAll() {
	c.receiveAll = true
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
		c.invokeCloseHooks()
	})
}

// AddCloseHook registers a callback that will be executed once when the client closes.
func (c *Client) AddCloseHook(fn func(*Client)) {
	if fn == nil {
		return
	}
	c.hookMu.Lock()
	defer c.hookMu.Unlock()
	c.closeHooks = append(c.closeHooks, fn)
}

func (c *Client) invokeCloseHooks() {
	c.hookMu.Lock()
	hooks := append([]func(*Client){}, c.closeHooks...)
	c.closeHooks = nil
	c.hookMu.Unlock()

	for _, hook := range hooks {
		func(h func(*Client)) {
			defer func() {
				if r := recover(); r != nil {
					slog.Warn("ws close hook panic", slog.Any("error", r))
				}
			}()
			h(c)
		}(hook)
	}
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
		c.processCommand(cmd)
	}
}

func (c *Client) processCommand(cmd Command) {
	if c.commands == nil {
		return
	}
	c.commands.Process(c, cmd)
}

type Hub struct {
	topics  map[string]map[*Client]struct{}
	clients map[string]*Client
	global  map[*Client]struct{}
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		topics:  make(map[string]map[*Client]struct{}),
		clients: make(map[string]*Client),
		global:  make(map[*Client]struct{}),
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
	if c.receiveAll {
		delete(h.global, c)
	}
	c.close()
	slog.Info("ws client detached", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID))
}

func (h *Hub) Broadcast(_ context.Context, msg *domain.Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("broadcast marshal error", slog.Any("error", err))
		return
	}
	h.mu.RLock()
	clientsMap := h.topics[msg.Topic]
	clients := make([]*Client, 0, len(clientsMap)+len(h.global))
	seen := make(map[*Client]struct{}, len(clientsMap)+len(h.global))
	for c := range clientsMap {
		clients = append(clients, c)
		seen[c] = struct{}{}
	}
	for c := range h.global {
		if _, ok := seen[c]; ok {
			continue
		}
		clients = append(clients, c)
	}
	h.mu.RUnlock()

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
		if trimmed := strings.TrimSpace(topic); trimmed != "" {
			h.subscribe(c, trimmed)
		}
	}
	slog.Info("ws client attached", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID), slog.String("sectionId", c.sectionID), slog.Any("topics", topics))
}

func normalizeAction(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}
