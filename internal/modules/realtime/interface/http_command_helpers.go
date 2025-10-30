package transport

import (
    "context"
    "encoding/json"
    "log/slog"
    "strings"
    "time"

    "mesaYaWs/internal/modules/realtime/application/usecase"
    domain "mesaYaWs/internal/modules/realtime/domain"
    "mesaYaWs/internal/modules/realtime/infrastructure"
)

type commandHandlerFactory func(entity, section, token string, connectUC *usecase.ConnectSectionUseCase) infrastructure.CommandHandler

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
    }
    for _, cfg := range configs {
        handlers[cfg.entity] = newGenericCommandHandler(cfg.entity, cfg.plural, cfg.singular)
    }
    return handlers
}()

func newGenericCommandHandler(canonicalEntity, pluralAction, singularAction string) commandHandlerFactory {
    normalizedPlural := strings.ToLower(strings.TrimSpace(pluralAction))
    normalizedSingular := strings.ToLower(strings.TrimSpace(singularAction))
    return func(entity, section, token string, connectUC *usecase.ConnectSectionUseCase) infrastructure.CommandHandler {
        return func(cmdCtx context.Context, client *infrastructure.Client, cmd infrastructure.Command) {
            action := strings.ToLower(strings.TrimSpace(cmd.Action))
            switch action {
            case "list_" + normalizedPlural, "list", "fetch_all":
                executeListCommand[domain.ListEntityCommand](cmdCtx, entity, section, token, cmd, client, connectUC.HandleListEntityCommand)
            case "get_" + normalizedSingular, "detail", "fetch_one":
                executeDetailCommand[domain.GetEntityCommand](cmdCtx, entity, section, token, cmd, client, connectUC.HandleGetEntityCommand, func(command domain.GetEntityCommand) string {
                    return command.ID
                })
            default:
                slog.Debug("ws handler generic unknown action", slog.String("entity", entity), slog.String("sectionId", section), slog.String("action", cmd.Action))
                sendCommandError(client, entity, section, "unknown", "unsupported action")
            }
        }
    }
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

func executeListCommand[T any](
    ctx context.Context,
    entity, section, token string,
    cmd infrastructure.Command,
    client *infrastructure.Client,
    listFn func(context.Context, string, string, T, string) (*domain.Message, error),
) {
    payload, err := decodeCommand[T](cmd.Payload)
    if err != nil {
        slog.Warn("ws handler list payload decode failed", slog.String("entity", entity), slog.String("sectionId", section), slog.Any("error", err))
        sendCommandError(client, entity, section, "list", "invalid payload")
        return
    }
    message, err := listFn(ctx, token, section, payload, entity)
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
    cmd infrastructure.Command,
    client *infrastructure.Client,
    detailFn func(context.Context, string, string, T, string) (*domain.Message, error),
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
    message, err := detailFn(ctx, token, section, payload, entity)
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
