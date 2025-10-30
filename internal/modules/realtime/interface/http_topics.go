package transport

import (
    "strings"

    domain "mesaYaWs/internal/modules/realtime/domain"
)

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
    case "auth-user", "auth-users", "auth_user", "auth_users":
        return "auth-users"
    case "authuser", "authusers", "auth":
        return "auth-users"
    default:
        return trimmed
    }
}
