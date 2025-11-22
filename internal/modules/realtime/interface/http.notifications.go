package transport

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"

	domain "mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/modules/realtime/infrastructure"
	"mesaYaWs/internal/shared/auth"
)

var notificationCounter atomic.Uint64

// NewNotificationsWebsocketHandler exposes /ws/notifications requiring authentication
// and streams broadcasted messages to the connected client.
func NewNotificationsWebsocketHandler(hub *infrastructure.Hub, validator auth.TokenValidator) func(echo.Context) error {
	return func(c echo.Context) error {
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		peerIP := c.RealIP()

		token := c.QueryParam("token")
		claims, err := validator.Validate(token)
		if err != nil {
			slog.Warn("notifications ws auth failed", slog.String("ip", peerIP), slog.Any("error", err))
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid or missing token")
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			slog.Error("notifications ws upgrade failed", slog.String("ip", peerIP), slog.String("reqID", requestID), slog.Any("error", err))
			return err
		}

		userID := claims.Subject
		sessionID := fmt.Sprintf("notif-%d", notificationCounter.Add(1))
		client := infrastructure.NewClient(hub, conn, userID, sessionID, "", "notifications", token, 8, nil)
		hub.AttachClientToAll(client)

		go client.WritePump()
		go client.ReadPump()

		connected := &domain.Message{
			Topic:  domain.TopicSystemConnected,
			Entity: domain.SystemEntity,
			Action: domain.ActionConnected,
			Metadata: map[string]string{
				"sessionId": sessionID,
				"userId":    userID,
			},
			Data: map[string]any{
				"mode":   "notifications",
				"topics": []string{"*"},
			},
			Timestamp: time.Now().UTC(),
		}
		client.SendDomainMessage(connected)

		slog.Info("notifications ws connected", slog.String("userId", userID), slog.String("sessionId", sessionID), slog.String("ip", peerIP), slog.String("reqID", requestID))
		return nil
	}
}
