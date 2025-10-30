package infrastructure

import (
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
	c.closeHooks = append(c.closeHooks, fn)
	c.hookMu.Unlock()
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
