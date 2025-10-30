package transport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/application/usecase"
	"mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/modules/realtime/infrastructure"
	"mesaYaWs/internal/shared/auth"
)

var analyticsCounter atomic.Uint64

// NewAnalyticsWebsocketHandler exposes /ws/analytics/:scope/:entity and streams analytics snapshots.
func NewAnalyticsWebsocketHandler(hub *infrastructure.Hub, analyticsUC *usecase.AnalyticsUseCase) func(echo.Context) error {
	return func(c echo.Context) error {
		scopeParam := c.Param("scope")
		entityParam := c.Param("entity")
		key := usecase.AnalyticsKey(scopeParam, entityParam)
		cfg, ok := analyticsUC.Endpoint(key)
		if !ok {
			slog.Warn("analytics ws unsupported endpoint", slog.String("scope", scopeParam), slog.String("entity", entityParam), slog.String("key", key))
			return echo.NewHTTPError(http.StatusNotFound, "analytics endpoint not available")
		}

		token := extractBearerToken(c.Request())
		request := cfg.RequestFromValues(c.QueryParams())

		ctx, cancel := context.WithTimeout(c.Request().Context(), 15*time.Second)
		defer cancel()

		output, err := analyticsUC.Connect(ctx, cfg.Key, token, request)
		if err != nil {
			status := http.StatusInternalServerError
			message := "unable to fetch analytics"

			switch {
			case errors.Is(err, usecase.ErrMissingToken), errors.Is(err, auth.ErrMissingToken):
				status = http.StatusBadRequest
				message = "missing token"
			case errors.Is(err, auth.ErrInvalidToken):
				status = http.StatusUnauthorized
				message = "invalid token"
			case errors.Is(err, usecase.ErrAnalyticsMissingIdentifier):
				status = http.StatusBadRequest
				message = "missing identifier"
			case errors.Is(err, port.ErrAnalyticsForbidden):
				status = http.StatusForbidden
				message = "forbidden"
			case errors.Is(err, port.ErrAnalyticsNotFound):
				status = http.StatusNotFound
				message = "analytics not found"
			case errors.Is(err, context.DeadlineExceeded):
				status = http.StatusGatewayTimeout
				message = "analytics timeout"
			default:
				slog.Error("analytics connect error", slog.String("key", cfg.Key), slog.Any("error", err))
			}

			return echo.NewHTTPError(status, message)
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			slog.Error("analytics ws upgrade failed", slog.String("key", cfg.Key), slog.Any("error", err))
			return err
		}

		var (
			userID    string
			sessionID string
			roles     []string
		)
		if claims := output.Claims; claims != nil {
			userID = strings.TrimSpace(claims.RegisteredClaims.Subject)
			sessionID = strings.TrimSpace(claims.SessionID)
			roles = claims.Roles
		}
		if sessionID == "" {
			sessionID = fmt.Sprintf("%s-%d", cfg.Key, analyticsCounter.Add(1))
		}

		topics := []string{domain.SnapshotTopic(cfg.Entity), domain.ErrorTopic(cfg.Entity)}
		baseRequest := output.Request.Clone()
		commandHandler := newAnalyticsCommandHandler(cfg.Key, cfg, analyticsUC, token, sessionID, &baseRequest)

		client := infrastructure.NewClient(hub, conn, userID, sessionID, "", cfg.Entity, token, 4, commandHandler)
		hub.AttachClient(client, topics)
		analyticsUC.RegisterSession(sessionID, cfg.Key, token, baseRequest)
		client.AddCloseHook(func(*infrastructure.Client) {
			analyticsUC.UnregisterSession(sessionID)
		})

		go client.WritePump()
		go client.ReadPump()

		if output.Message != nil {
			client.SendDomainMessage(output.Message)
		}

		metadata := map[string]string{
			"scope":        cfg.Scope,
			"analyticsKey": cfg.Key,
		}
		if userID != "" {
			metadata["userId"] = userID
		}
		if sessionID != "" {
			metadata["sessionId"] = sessionID
		}

		payload := map[string]any{
			"mode":       "analytics",
			"entity":     cfg.Entity,
			"topics":     topics,
			"identifier": output.Request.Identifier,
			"query":      output.Request.Query,
			"roles":      roles,
		}

		connected := &domain.Message{
			Topic:     domain.TopicSystemConnected,
			Entity:    domain.SystemEntity,
			Action:    domain.ActionConnected,
			Metadata:  metadata,
			Data:      payload,
			Timestamp: time.Now().UTC(),
		}
		client.SendDomainMessage(connected)
		slog.Info("analytics ws connected", slog.String("key", cfg.Key), slog.String("userId", userID), slog.String("sessionId", sessionID), slog.String("scope", cfg.Scope))

		return nil
	}
}

func extractBearerToken(r *http.Request) string {
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	lower := strings.ToLower(authz)
	if strings.HasPrefix(lower, "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	return ""
}

func newAnalyticsCommandHandler(key string, cfg usecase.AnalyticsEndpointConfig, analyticsUC *usecase.AnalyticsUseCase, token, sessionID string, base *domain.AnalyticsRequest) infrastructure.CommandHandler {
	trimmedToken := strings.TrimSpace(token)
	return func(cmdCtx context.Context, client *infrastructure.Client, cmd infrastructure.Command) {
		action := strings.ToLower(strings.TrimSpace(cmd.Action))
		switch action {
		case "refresh", "fetch", "query":
			payload, err := decodeCommand[domain.AnalyticsCommand](cmd.Payload)
			if err != nil {
				slog.Warn("analytics command decode failed", slog.String("key", key), slog.String("action", action), slog.Any("error", err))
				sendCommandError(client, cfg.Entity, "", action, "invalid payload")
				return
			}

			commandCtx, cancel := context.WithTimeout(cmdCtx, 15*time.Second)
			defer cancel()

			message, updated, err := analyticsUC.HandleCommand(commandCtx, key, trimmedToken, *base, payload)
			if err != nil {
				slog.Warn("analytics command failed", slog.String("key", key), slog.String("action", action), slog.Any("error", err))
				sendCommandError(client, cfg.Entity, "", action, err.Error())
				return
			}

			*base = updated.Clone()
			analyticsUC.UpdateSession(sessionID, key, trimmedToken, *base)
			client.SendDomainMessage(message)
		default:
			slog.Debug("analytics command unsupported", slog.String("key", key), slog.String("action", action))
			sendCommandError(client, cfg.Entity, "", action, "unsupported action")
		}
	}
}
