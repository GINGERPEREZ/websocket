package transport

import (
	"net/http"
	"strings"
	"time"

	"mesaYaWs/internal/realtime/domain"
	"mesaYaWs/internal/realtime/infrastructure"
	"mesaYaWs/internal/shared/auth"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWebsocketHandler crea un handler que expone la ruta /ws/restaurant/:section/:token
// donde :section es el namespace/section y :token es el JWT emitido por el servicio Nest.
// Valida el JWT localmente con el validador proporcionado, registra al cliente en el hub y
// responde con un evento system.connected.
func NewWebsocketHandler(
	hub *infrastructure.Hub,
	validator auth.TokenValidator,
	entity string,
	allowedActions []string,
) func(echo.Context) error {
	entity = strings.TrimSpace(entity)
	if entity == "" {
		entity = "restaurants"
	}
	if len(allowedActions) == 0 {
		allowedActions = []string{"created", "updated", "deleted", "snapshot"}
	}

	return func(c echo.Context) error {
		section := strings.TrimSpace(c.Param("section"))
		token := strings.TrimSpace(c.Param("token"))
		logger := c.Logger()
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		peerIP := c.RealIP()

		if section == "" {
			logger.Warnf("ws rejected: missing section entity=%s ip=%s reqID=%s", entity, peerIP, requestID)
			return echo.NewHTTPError(http.StatusBadRequest, "missing section")
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			logger.Errorf("ws upgrade failed entity=%s section=%s ip=%s reqID=%s: %v", entity, section, peerIP, requestID, err)
			return err
		}

		claims, err := validator.Validate(token)
		if err != nil {
			logger.Warnf("ws token invalid entity=%s section=%s ip=%s reqID=%s: %v", entity, section, peerIP, requestID, err)
			_ = conn.Close()
			return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
		}

		userID := claims.RegisteredClaims.Subject
		sessionID := claims.SessionID
		roles := claims.Roles

		client := infrastructure.NewClient(hub, conn, userID, sessionID, section, 8)

		topics := buildTopics(entity, allowedActions)
		hub.AttachClient(client, topics)

		go client.WritePump()
		go client.ReadPump()

		connected := &domain.Message{
			Topic:  "system.connected",
			Entity: "system",
			Action: "connected",
			Metadata: map[string]string{
				"userId":    userID,
				"sessionId": sessionID,
				"sectionId": section,
			},
			Data: map[string]interface{}{
				"entity":        entity,
				"sectionId":     section,
				"allowedTopics": topics,
				"roles":         roles,
			},
			Timestamp: time.Now().UTC(),
		}
		client.SendDomainMessage(connected)

		logger.Infof("ws connected entity=%s section=%s user=%s session=%s roles=%v ip=%s reqID=%s",
			entity, section, userID, sessionID, roles, peerIP, requestID)

		return nil
	}
}

func buildTopics(entity string, allowedActions []string) []string {
	topics := []string{entity + ".snapshot"}
	seen := map[string]struct{}{
		topics[0]: {},
	}
	for _, action := range allowedActions {
		action = strings.TrimSpace(strings.ToLower(action))
		if action == "" {
			continue
		}
		topic := entity + "." + action
		if _, exists := seen[topic]; exists {
			continue
		}
		topics = append(topics, topic)
		seen[topic] = struct{}{}
	}
	return topics
}
