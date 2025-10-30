package transport

import (
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/labstack/echo/v4"

	domain "mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/modules/realtime/infrastructure"
)

var notificationCounter atomic.Uint64

// NewNotificationsWebsocketHandler exposes /ws/notifications without requiring authentication
// and streams every broadcasted message to the connected client.
func NewNotificationsWebsocketHandler(hub *infrastructure.Hub) func(echo.Context) error {
	return func(c echo.Context) error {
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		peerIP := c.RealIP()

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			slog.Error("notifications ws upgrade failed", slog.String("ip", peerIP), slog.String("reqID", requestID), slog.Any("error", err))
			return err
		}

		sessionID := fmt.Sprintf("notif-%d", notificationCounter.Add(1))
		client := infrastructure.NewClient(hub, conn, "notifications", sessionID, "", "notifications", "", 8, nil)
		hub.AttachClientToAll(client)

		go client.WritePump()
		go client.ReadPump()

		connected := &domain.Message{
			Topic:  domain.TopicSystemConnected,
			Entity: domain.SystemEntity,
			Action: domain.ActionConnected,
			Metadata: map[string]string{
				"sessionId": sessionID,
			},
			Data: map[string]any{
				"mode":   "notifications",
				"topics": []string{"*"},
			},
			Timestamp: time.Now().UTC(),
		}
		client.SendDomainMessage(connected)

		slog.Info("notifications ws connected", slog.String("sessionId", sessionID), slog.String("ip", peerIP), slog.String("reqID", requestID))
		return nil
	}
}
