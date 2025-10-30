package transport

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"mesaYaWs/internal/modules/realtime/application/port"
	"mesaYaWs/internal/modules/realtime/application/usecase"
	domain "mesaYaWs/internal/modules/realtime/domain"
	"mesaYaWs/internal/modules/realtime/infrastructure"
	"mesaYaWs/internal/shared/auth"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// NewWebsocketHandler expone /ws/:entity/:section/:token y valida el JWT localmente.
func NewWebsocketHandler(
	hub *infrastructure.Hub,
	connectUC *usecase.ConnectSectionUseCase,
	defaultEntity string,
	allowedActions []string,
) func(echo.Context) error {
	defaultEntity = normalizeEntity(defaultEntity)
	if defaultEntity == "" {
		defaultEntity = "restaurants"
	}
	if len(allowedActions) == 0 {
		allowedActions = []string{"created", "updated", "deleted", "snapshot"}
	}

	return func(c echo.Context) error {
		entityParam := c.Param("entity")
		entity := normalizeEntity(entityParam)
		if entity == "" {
			entity = defaultEntity
		}
		section := strings.TrimSpace(c.Param("section"))
		token := strings.TrimSpace(c.Param("token"))
		queryParams := c.QueryParams()
		if token == "" {
			token = strings.TrimSpace(queryParams.Get("token"))
			if token != "" {
				slog.Debug("ws handler token sourced from query", slog.String("entity", entity), slog.String("sectionId", section), slog.Int("tokenLen", len(token)))
			}
		}
		if token == "" {
			authz := strings.TrimSpace(c.Request().Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				token = strings.TrimSpace(authz[7:])
				slog.Debug("ws handler token sourced from authorization header", slog.String("entity", entity), slog.String("sectionId", section), slog.Int("tokenLen", len(token)))
			}
		}
		logger := c.Logger()
		requestID := c.Response().Header().Get(echo.HeaderXRequestID)
		peerIP := c.RealIP()

		if entity == "" {
			slog.Warn("ws handler missing entity", slog.String("sectionId", section))
			logger.Warnf("ws rejected: missing entity section=%s ip=%s reqID=%s", section, peerIP, requestID)
			return echo.NewHTTPError(http.StatusBadRequest, "missing entity")
		}

		factory, supported := entityHandlers[entity]
		if !supported {
			slog.Warn("ws handler entity not integrated", slog.String("entity", entity), slog.String("sectionId", section))
			logger.Warnf("ws rejected: entity not integrated entity=%s section=%s ip=%s reqID=%s", entity, section, peerIP, requestID)
			return echo.NewHTTPError(http.StatusNotFound, "entity "+entity+" is not integrated")
		}

		if section == "" {
			slog.Warn("ws handler missing section", slog.String("entity", entity), slog.Int("tokenLen", len(token)))
			logger.Warnf("ws rejected: missing section entity=%s ip=%s reqID=%s", entity, peerIP, requestID)
			return echo.NewHTTPError(http.StatusBadRequest, "missing section")
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
		defer cancel()

		slog.Info("ws handler executing connect", slog.String("entity", entity), slog.String("sectionId", section), slog.Int("tokenLen", len(token)))
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

			slog.Warn("ws handler connect failed", slog.String("entity", entity), slog.String("sectionId", section), slog.Int("status", status), slog.String("message", message), slog.Any("error", err))
			if status >= http.StatusInternalServerError {
				logger.Errorf("ws connect failed entity=%s section=%s ip=%s reqID=%s: %v", entity, section, peerIP, requestID, err)
			} else {
				logger.Warnf("ws connect rejected entity=%s section=%s ip=%s reqID=%s: %v", entity, section, peerIP, requestID, err)
			}
			return echo.NewHTTPError(status, message)
		}

		conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
		if err != nil {
			slog.Error("ws handler upgrade failed", slog.String("entity", entity), slog.String("sectionId", section), slog.Any("error", err))
			logger.Errorf("ws upgrade failed entity=%s section=%s ip=%s reqID=%s: %v", entity, section, peerIP, requestID, err)
			return err
		}

		claims := output.Claims
		userID := claims.RegisteredClaims.Subject
		sessionID := claims.SessionID
		roles := claims.Roles
		slog.Info("ws handler upgrade success", slog.String("entity", entity), slog.String("sectionId", section), slog.String("userId", userID), slog.String("sessionId", sessionID), slog.Any("roles", roles))

		commandHandler := factory(entity, section, token, connectUC)

		client := infrastructure.NewClient(hub, conn, userID, sessionID, section, entity, token, 8, commandHandler)

		topics := buildTopics(entity, allowedActions)
		hub.AttachClient(client, topics)

		go client.WritePump()
		go client.ReadPump()

		connected := &domain.Message{
			Topic:  domain.TopicSystemConnected,
			Entity: domain.SystemEntity,
			Action: domain.ActionConnected,
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
		slog.Info("ws handler sent system.connected", slog.String("entity", entity), slog.String("sectionId", section), slog.String("userId", userID), slog.String("sessionId", sessionID))

		logger.Infof("ws connected entity=%s section=%s user=%s session=%s roles=%v ip=%s reqID=%s",
			entity, section, userID, sessionID, roles, peerIP, requestID)

		return nil
	}
}

