package infrastructure

import (
	"context"
	"encoding/json"
	"log/slog"
	"mesaYaWs/internal/modules/realtime/domain"
	"strings"
	"sync"
)

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

// AttachClientToAll registers the client as a global subscriber receiving every broadcasted message.
func (h *Hub) AttachClientToAll(c *Client) {
	c.EnableReceiveAll()
	h.registerClient(c)
	h.mu.Lock()
	h.global[c] = struct{}{}
	h.mu.Unlock()
	slog.Info("ws client attached to all topics", slog.String("userId", c.userID), slog.String("sessionId", c.sessionID))
}
