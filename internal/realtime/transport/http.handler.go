package transport

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"mesaYaWs/internal/realtime/application/port"
	"mesaYaWs/internal/realtime/application/usecase"
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
	connectUC *usecase.ConnectSectionUseCase,
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
		if token == "" {
			token = strings.TrimSpace(c.QueryParam("token"))
			if token != "" {
				log.Printf("ws handler: token sourced from query section=%s tokenLen=%d", section, len(token))
			}
		}
		if token == "" {
			authz := strings.TrimSpace(c.Request().Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				token = strings.TrimSpace(authz[7:])
				log.Printf("ws handler: token sourced from authorization header section=%s tokenLen=%d", section, len(token))
			}
		}
		logger := c.Logger()
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		peerIP := c.RealIP()

		if section == "" {
			log.Printf("ws handler: missing section path param tokenLen=%d", len(token))
			logger.Warnf("ws rejected: missing section entity=%s ip=%s reqID=%s", entity, peerIP, requestID)
			return echo.NewHTTPError(http.StatusBadRequest, "missing section")
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
		defer cancel()

		log.Printf("ws handler: executing connect usecase section=%s tokenLen=%d", section, len(token))
		output, err := connectUC.Execute(ctx, usecase.ConnectSectionInput{Token: token, SectionID: section})
		if err != nil {
			status := http.StatusInternalServerError
			message := "unable to connect section"

			switch {
			case errors.Is(err, usecase.ErrMissingToken), errors.Is(err, auth.ErrMissingToken):
				status = http.StatusBadRequest
				message = "missing token"
			case errors.Is(err, usecase.ErrMissingSection):
				status = http.StatusBadRequest
				message = "missing section"
			case errors.Is(err, auth.ErrInvalidToken):
				status = http.StatusUnauthorized
				message = "invalid token"
			case errors.Is(err, port.ErrSnapshotForbidden):
				status = http.StatusForbidden
				message = "forbidden"
			case errors.Is(err, port.ErrSnapshotNotFound):
				status = http.StatusNotFound
				message = "section not found"
			case errors.Is(err, context.DeadlineExceeded):
				status = http.StatusGatewayTimeout
				message = "snapshot timeout"
			}

			log.Printf("ws handler: connect failed section=%s status=%d message=%s err=%v", section, status, message, err)
			if status >= http.StatusInternalServerError {
				logger.Errorf("ws connect failed entity=%s section=%s ip=%s reqID=%s: %v", entity, section, peerIP, requestID, err)
			} else {
				logger.Warnf("ws connect rejected entity=%s section=%s ip=%s reqID=%s: %v", entity, section, peerIP, requestID, err)
			}
			return echo.NewHTTPError(status, message)
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			log.Printf("ws handler: upgrade failed section=%s err=%v", section, err)
			logger.Errorf("ws upgrade failed entity=%s section=%s ip=%s reqID=%s: %v", entity, section, peerIP, requestID, err)
			return err
		}

		claims := output.Claims
		userID := claims.RegisteredClaims.Subject
		sessionID := claims.SessionID
		roles := claims.Roles
		log.Printf("ws handler: upgrade success section=%s user=%s session=%s roles=%v", section, userID, sessionID, roles)

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
		log.Printf("ws handler: sent system.connected section=%s user=%s session=%s", section, userID, sessionID)

		if output.Snapshot != nil {
			snapshot := &domain.Message{
				Topic:      entity + ".snapshot",
				Entity:     entity,
				Action:     "snapshot",
				ResourceID: section,
				Metadata: map[string]string{
					"userId":    userID,
					"sessionId": sessionID,
					"sectionId": section,
				},
				Data:      output.Snapshot.Payload,
				Timestamp: time.Now().UTC(),
			}
			client.SendDomainMessage(snapshot)
			log.Printf("ws handler: sent snapshot section=%s user=%s session=%s payloadType=%T", section, userID, sessionID, output.Snapshot.Payload)
		}

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
