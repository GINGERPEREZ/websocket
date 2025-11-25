package transport

import (
	"context"
	"encoding/json"
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

const (
	roleAdmin = "ADMIN"
	roleOwner = "OWNER"
	roleUser  = "USER"
)

type entityPolicy struct {
	allowedRoles map[string]struct{}
}

func newEntityPolicy(roles ...string) entityPolicy {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		trimmed := strings.ToUpper(strings.TrimSpace(role))
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	return entityPolicy{allowedRoles: allowed}
}

func (p entityPolicy) allows(role string) bool {
	if len(p.allowedRoles) == 0 {
		return true
	}
	normalized := strings.ToUpper(strings.TrimSpace(role))
	_, ok := p.allowedRoles[normalized]
	return ok
}

var entityPolicies = map[string]entityPolicy{
	"restaurants":        newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"tables":             newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"reservations":       newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"reviews":            newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"sections":           newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"objects":            newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"menus":              newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"dishes":             newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"images":             newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"section-objects":    newEntityPolicy(roleAdmin, roleOwner),
	"payments":           newEntityPolicy(roleAdmin, roleOwner),
	"subscriptions":      newEntityPolicy(roleAdmin, roleOwner),
	"subscription-plans": newEntityPolicy(roleAdmin, roleOwner, roleUser),
	"users":              newEntityPolicy(roleAdmin),
	"owner-upgrades":     newEntityPolicy(roleAdmin),
}

func isEntityAccessAllowed(entity string, claims *auth.Claims) bool {
	if claims == nil {
		return false
	}
	policy, present := entityPolicies[entity]
	if !present {
		return true
	}
	for _, role := range claims.Roles {
		if policy.allows(role) {
			return true
		}
	}
	return false
}

func snapshotAudienceFromClaims(claims *auth.Claims) port.SnapshotAudience {
	if claims == nil {
		return port.SnapshotAudienceUser
	}
	audience := port.SnapshotAudienceUser
	for _, role := range claims.Roles {
		normalized := strings.ToUpper(strings.TrimSpace(role))
		switch normalized {
		case roleAdmin:
			return port.SnapshotAudienceAdmin
		case roleOwner:
			audience = port.SnapshotAudienceOwner
		}
	}
	return audience
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

		if !isEntityAccessAllowed(entity, output.Claims) {
			roles := []string{}
			if output.Claims != nil {
				roles = append(roles, output.Claims.Roles...)
			}
			slog.Warn("ws handler forbidden entity access", slog.String("entity", entity), slog.String("sectionId", section), slog.Any("roles", roles))
			logger.Warnf("ws rejected: forbidden entity entity=%s section=%s roles=%v ip=%s reqID=%s", entity, section, roles, peerIP, requestID)
			return echo.NewHTTPError(http.StatusForbidden, "forbidden")
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

		commandHandler := factory(entity, section, token, output.Claims, connectUC)

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

func buildTopics(entity string, allowedActions []string) []string {
	entity = strings.TrimSpace(entity)
	baseTopics := []string{
		domain.SnapshotTopic(entity),
		domain.ListTopic(entity),
		domain.DetailTopic(entity),
		domain.ErrorTopic(entity),
	}
	topics := make([]string, 0, len(baseTopics)+len(allowedActions))
	seen := make(map[string]struct{}, len(baseTopics)+len(allowedActions))
	for _, topic := range baseTopics {
		if topic == "" {
			continue
		}
		topics = append(topics, topic)
		seen[topic] = struct{}{}
	}
	for _, action := range allowedActions {
		action = strings.TrimSpace(strings.ToLower(action))
		if action == "" {
			continue
		}
		topic := domain.CustomTopic(entity, action)
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
		Topic:      domain.ErrorTopic(entity),
		Entity:     entity,
		Action:     domain.ActionError,
		ResourceID: section,
		Metadata:   metadata,
		Data: map[string]string{
			"error": reason,
		},
		Timestamp: time.Now().UTC(),
	}
	client.SendDomainMessage(message)
}

type commandHandlerFactory func(entity, section, token string, claims *auth.Claims, connectUC *usecase.ConnectSectionUseCase) func(context.Context, *infrastructure.Client, infrastructure.Command)

var entityHandlers = func() map[string]commandHandlerFactory {
	handlers := make(map[string]commandHandlerFactory)
	configs := []struct {
		entity   string
		plural   string
		singular string
	}{
		{entity: "restaurants", plural: "restaurants", singular: "restaurant"},
		{entity: "tables", plural: "tables", singular: "table"},
		{entity: "reservations", plural: "reservations", singular: "reservation"},
		{entity: "reviews", plural: "reviews", singular: "review"},
		{entity: "sections", plural: "sections", singular: "section"},
		{entity: "objects", plural: "objects", singular: "object"},
		{entity: "menus", plural: "menus", singular: "menu"},
		{entity: "dishes", plural: "dishes", singular: "dish"},
		{entity: "images", plural: "images", singular: "image"},
		{entity: "section-objects", plural: "section_objects", singular: "section_object"},
		{entity: "payments", plural: "payments", singular: "payment"},
		{entity: "subscriptions", plural: "subscriptions", singular: "subscription"},
		{entity: "subscription-plans", plural: "subscription_plans", singular: "subscription_plan"},
		{entity: "auth-users", plural: "auth_users", singular: "auth_user"},
		{entity: "owners", plural: "owners", singular: "owner"},
		{entity: "owner-upgrade", plural: "owner_upgrade_requests", singular: "owner_upgrade_request"},
	}
	for _, cfg := range configs {
		handlers[cfg.entity] = newGenericCommandHandler(cfg.entity, cfg.plural, cfg.singular)
	}
	return handlers
}()

func newGenericCommandHandler(canonicalEntity, pluralAction, singularAction string) commandHandlerFactory {
	normalizedPlural := strings.ToLower(strings.TrimSpace(pluralAction))
	normalizedSingular := strings.ToLower(strings.TrimSpace(singularAction))
	return func(entity, section, token string, claims *auth.Claims, connectUC *usecase.ConnectSectionUseCase) func(context.Context, *infrastructure.Client, infrastructure.Command) {
		snapshotCtx := port.SnapshotContext{
			SectionID: strings.TrimSpace(section),
			Audience:  snapshotAudienceFromClaims(claims),
		}
		return func(cmdCtx context.Context, client *infrastructure.Client, cmd infrastructure.Command) {
			action := strings.ToLower(strings.TrimSpace(cmd.Action))
			switch action {
			case "list_" + normalizedPlural, "list", "fetch_all":
				executeListCommand[domain.ListEntityCommand](cmdCtx, entity, section, token, snapshotCtx, cmd, client, connectUC.HandleListEntityCommand)
			case "get_" + normalizedSingular, "detail", "fetch_one":
				executeDetailCommand[domain.GetEntityCommand](cmdCtx, entity, section, token, snapshotCtx, cmd, client, connectUC.HandleGetEntityCommand, func(command domain.GetEntityCommand) string {
					return command.ID
				})
			default:
				slog.Debug("ws handler generic unknown action", slog.String("entity", entity), slog.String("sectionId", section), slog.String("action", cmd.Action))
				sendCommandError(client, entity, section, "unknown", "unsupported action")
			}
		}
	}
}

func normalizeEntity(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch trimmed {
	case "", "-", "default":
		return ""
	case "restaurant", "restaurants":
		return "restaurants"
	case "table", "tables":
		return "tables"
	case "reservation", "reservations":
		return "reservations"
	case "section", "sections":
		return "sections"
	case "review", "reviews":
		return "reviews"
	case "sectionobject", "sectionobjects":
		return "section-objects"
	case "object", "objects":
		return "objects"
	case "menu", "menus":
		return "menus"
	case "dish", "dishes":
		return "dishes"
	case "image", "images":
		return "images"
	case "section-object", "section-objects", "section_object", "section_objects":
		return "section-objects"
	case "payment", "payments":
		return "payments"
	case "subscription", "subscriptions":
		return "subscriptions"
	case "subscription-plan", "subscription-plans", "subscription_plan", "subscription_plans":
		return "subscription-plans"
	case "subscriptionplan", "subscriptionplans":
		return "subscription-plans"
	case "user", "users", "auth-user", "auth-users", "auth_user", "auth_users", "authuser", "authusers", "auth":
		return "users"
	case "owner", "owners":
		return "users"
	case "owner-upgrade", "owner-upgrades", "owner_upgrade", "owner_upgrades", "ownerupgrade", "ownerupgrades":
		return "owner-upgrades"
	default:
		return trimmed
	}
}

func executeListCommand[T any](
	ctx context.Context,
	entity, section, token string,
	snapshotCtx port.SnapshotContext,
	cmd infrastructure.Command,
	client *infrastructure.Client,
	listFn func(context.Context, string, port.SnapshotContext, T, string) (*domain.Message, error),
) {
	slog.Debug("ws handler list command received", slog.String("entity", entity), slog.String("sectionId", section), slog.String("rawPayload", string(cmd.Payload)))
	payload, err := decodeCommand[T](cmd.Payload)
	if err != nil {
		slog.Warn("ws handler list payload decode failed", slog.String("entity", entity), slog.String("sectionId", section), slog.Any("error", err))
		sendCommandError(client, entity, section, "list", "invalid payload")
		return
	}
	slog.Debug("ws handler list command decoded", slog.String("entity", entity), slog.String("sectionId", section), slog.Any("payload", payload))
	message, err := listFn(ctx, token, snapshotCtx, payload, entity)
	if err != nil {
		slog.Warn("ws handler list fetch failed", slog.String("entity", entity), slog.String("sectionId", section), slog.Any("error", err))
		sendCommandError(client, entity, section, "list", err.Error())
		return
	}
	client.SendDomainMessage(message)
}

func executeDetailCommand[T any](
	ctx context.Context,
	entity, section, token string,
	snapshotCtx port.SnapshotContext,
	cmd infrastructure.Command,
	client *infrastructure.Client,
	detailFn func(context.Context, string, port.SnapshotContext, T, string) (*domain.Message, error),
	resourceExtractor func(T) string,
) {
	payload, err := decodeCommand[T](cmd.Payload)
	if err != nil {
		slog.Warn("ws handler detail payload decode failed", slog.String("entity", entity), slog.String("sectionId", section), slog.Any("error", err))
		sendCommandError(client, entity, section, "detail", "invalid payload")
		return
	}
	resourceID := strings.TrimSpace(resourceExtractor(payload))
	if resourceID == "" {
		sendCommandError(client, entity, section, "detail", "invalid payload")
		return
	}
	message, err := detailFn(ctx, token, snapshotCtx, payload, entity)
	if err != nil {
		slog.Warn("ws handler detail fetch failed", slog.String("entity", entity), slog.String("sectionId", section), slog.String("resourceId", resourceID), slog.Any("error", err))
		sendCommandError(client, entity, section, "detail", err.Error())
		return
	}
	client.SendDomainMessage(message)
}

func decodeCommand[T any](raw json.RawMessage) (T, error) {
	var payload T
	if len(raw) == 0 {
		return payload, nil
	}
	return payload, json.Unmarshal(raw, &payload)
}
