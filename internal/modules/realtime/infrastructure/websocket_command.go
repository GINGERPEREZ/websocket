package infrastructure

import (
	"context"
	"encoding/json"
	"log/slog"
	"mesaYaWs/internal/modules/realtime/domain"
	"strings"
	"time"
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

func normalizeAction(action string) string {
	return strings.ToLower(strings.TrimSpace(action))
}
