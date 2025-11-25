package transport

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/labstack/echo/v4"

	"mesaYaWs/internal/modules/realtime/infrastructure"
	"mesaYaWs/internal/shared/auth"
)

var notificationCounter atomic.Uint64

// allowedNotificationTopics define los topics que un usuario autenticado puede recibir
// como notificaciones. Esto evita enviar mensajes del sistema o de sincronización.
var allowedNotificationTopics = []string{
	// Eventos de entidades que generan notificaciones significativas para el usuario
	"reservations.created",
	"reservations.updated",
	"reservations.deleted",
	"reservations.status-changed",
	"restaurants.created",
	"restaurants.updated",
	"restaurants.deleted",
	"tables.created",
	"tables.updated",
	"tables.deleted",
	"sections.created",
	"sections.updated",
	"sections.deleted",
	"menus.created",
	"menus.updated",
	"menus.deleted",
	"dishes.created",
	"dishes.updated",
	"dishes.deleted",
	"reviews.created",
	"reviews.updated",
	"reviews.deleted",
	"payments.created",
	"payments.updated",
	"payments.status-changed",
	"subscriptions.created",
	"subscriptions.updated",
	"subscriptions.status-changed",
}

// roleBasedTopicFilter filtra topics según el rol del usuario
func roleBasedTopicFilter(topics []string, roles []string) []string {
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[strings.ToUpper(strings.TrimSpace(r))] = struct{}{}
	}

	_, isAdmin := roleSet["ADMIN"]
	_, isOwner := roleSet["OWNER"]

	filtered := make([]string, 0, len(topics))
	for _, topic := range topics {
		// Admin recibe todo
		if isAdmin {
			filtered = append(filtered, topic)
			continue
		}
		// Owner recibe la mayoría excepto topics sensibles de admin
		if isOwner {
			// Excluir topics de administración de usuarios
			if strings.HasPrefix(topic, "users.") || strings.HasPrefix(topic, "owner-upgrades.") {
				continue
			}
			filtered = append(filtered, topic)
			continue
		}
		// Usuario normal solo recibe topics de reservaciones, reviews y restaurantes públicos
		if strings.HasPrefix(topic, "reservations.") ||
			strings.HasPrefix(topic, "reviews.") ||
			strings.HasPrefix(topic, "restaurants.") {
			filtered = append(filtered, topic)
		}
	}
	return filtered
}

// NewNotificationsWebsocketHandler exposes /ws/notifications requiring authentication
// and streams broadcasted messages to the connected client.
// Solo envía notificaciones relevantes según el rol del usuario.
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
		roles := claims.Roles

		// Filtrar topics según el rol del usuario
		filteredTopics := roleBasedTopicFilter(allowedNotificationTopics, roles)

		client := infrastructure.NewClient(hub, conn, userID, sessionID, "", "notifications", token, 8, nil)
		// Suscribir solo a topics filtrados por rol en lugar de todos
		hub.AttachClient(client, filteredTopics)

		go client.WritePump()
		go client.ReadPump()

		// No enviar mensaje de sistema "connected" - es ruido innecesario para el cliente
		// El cliente sabe que está conectado por el éxito del handshake WebSocket

		slog.Info("notifications ws connected",
			slog.String("userId", userID),
			slog.String("sessionId", sessionID),
			slog.Any("roles", roles),
			slog.Int("topicCount", len(filteredTopics)),
			slog.String("ip", peerIP),
			slog.String("reqID", requestID))
		return nil
	}
}
