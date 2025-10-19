package transport

import (
	"context"
	"encoding/json"
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
		queryParams := c.QueryParams()
		if token == "" {
			token = strings.TrimSpace(queryParams.Get("token"))
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

		commandHandler := func(cmdCtx context.Context, client *infrastructure.Client, cmd infrastructure.Command) {
			action := strings.ToLower(strings.TrimSpace(cmd.Action))
			switch action {
			case "list_restaurants", "fetch_all", "list":
				var payload struct {
					Page      int    `json:"page"`
					Limit     int    `json:"limit"`
					Search    string `json:"search"`
					SortBy    string `json:"sortBy"`
					SortOrder string `json:"sortOrder"`
				}
				if len(cmd.Payload) > 0 {
					if err := json.Unmarshal(cmd.Payload, &payload); err != nil {
						log.Printf("ws handler: list payload decode failed section=%s err=%v", section, err)
						sendCommandError(client, entity, section, "list", "invalid payload")
						return
					}
				}
				params := domain.PagedQuery{
					Page:      payload.Page,
					Limit:     payload.Limit,
					Search:    payload.Search,
					SortBy:    payload.SortBy,
					SortOrder: payload.SortOrder,
				}
				snapshot, normalized, err := connectUC.ListRestaurants(cmdCtx, token, section, params)
				if err != nil {
					log.Printf("ws handler: list fetch failed section=%s err=%v", section, err)
					sendCommandError(client, entity, section, "list", err.Error())
					return
				}
				metadata := normalized.Metadata(section)
				message := &domain.Message{
					Topic:      entity + ".list",
					Entity:     entity,
					Action:     "list",
					ResourceID: section,
					Metadata:   metadata,
					Data:       snapshot.Payload,
					Timestamp:  time.Now().UTC(),
				}
				client.SendDomainMessage(message)
			case "get_restaurant", "fetch_one", "detail":
				var payload struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(cmd.Payload, &payload); err != nil || strings.TrimSpace(payload.ID) == "" {
					log.Printf("ws handler: detail payload decode failed section=%s err=%v", section, err)
					sendCommandError(client, entity, section, "detail", "invalid payload")
					return
				}
				snapshot, err := connectUC.GetRestaurant(cmdCtx, token, section, payload.ID)
				if err != nil {
					log.Printf("ws handler: detail fetch failed section=%s restaurant=%s err=%v", section, payload.ID, err)
					sendCommandError(client, entity, section, "detail", err.Error())
					return
				}
				metadata := map[string]string{
					"sectionId":    section,
					"restaurantId": strings.TrimSpace(payload.ID),
				}
				message := &domain.Message{
					Topic:      entity + ".detail",
					Entity:     entity,
					Action:     "detail",
					ResourceID: strings.TrimSpace(payload.ID),
					Metadata:   metadata,
					Data:       snapshot.Payload,
					Timestamp:  time.Now().UTC(),
				}
				client.SendDomainMessage(message)
			default:
				log.Printf("ws handler: unknown action section=%s action=%s", section, cmd.Action)
				sendCommandError(client, entity, section, "unknown", "unsupported action")
			}
		}

		client := infrastructure.NewClient(hub, conn, userID, sessionID, section, entity, token, 8, commandHandler)

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

		logger.Infof("ws connected entity=%s section=%s user=%s session=%s roles=%v ip=%s reqID=%s",
			entity, section, userID, sessionID, roles, peerIP, requestID)

		return nil
	}
}

func buildTopics(entity string, allowedActions []string) []string {
	topics := []string{entity + ".snapshot", entity + ".list", entity + ".detail", entity + ".error"}
	seen := map[string]struct{}{
		topics[0]: {},
		topics[1]: {},
		topics[2]: {},
		topics[3]: {},
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

func sendCommandError(client *infrastructure.Client, entity, section, action, reason string) {
	metadata := map[string]string{
		"sectionId": section,
		"action":    action,
	}
	if strings.TrimSpace(reason) != "" {
		metadata["reason"] = reason
	}
	message := &domain.Message{
		Topic:      entity + ".error",
		Entity:     entity,
		Action:     "error",
		ResourceID: section,
		Metadata:   metadata,
		Data: map[string]string{
			"error": reason,
		},
		Timestamp: time.Now().UTC(),
	}
	client.SendDomainMessage(message)
}
