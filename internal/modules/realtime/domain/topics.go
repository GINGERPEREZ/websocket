package domain

import "strings"

const (
	SystemEntity = "system"

	TopicSystemConnected = SystemEntity + ".connected"
	TopicSystemPong      = SystemEntity + ".pong"
	TopicSystemError     = SystemEntity + ".error"

	ActionConnected = "connected"
	ActionPong      = "pong"
	ActionError     = "error"
	ActionList      = "list"
	ActionDetail    = "detail"
	ActionSnapshot  = "snapshot"
	ActionCreated   = "created"
	ActionUpdated   = "updated"
	ActionDeleted   = "deleted"
)

// SnapshotTopic returns the canonical snapshot topic for the given entity.
func SnapshotTopic(entity string) string {
	return buildEntityTopic(entity, ActionSnapshot)
}

// ListTopic returns the canonical list topic for the given entity.
func ListTopic(entity string) string {
	return buildEntityTopic(entity, ActionList)
}

// DetailTopic returns the canonical detail topic for the given entity.
func DetailTopic(entity string) string {
	return buildEntityTopic(entity, ActionDetail)
}

// ErrorTopic returns the canonical error topic for the given entity.
func ErrorTopic(entity string) string {
	return buildEntityTopic(entity, ActionError)
}

// CreatedTopic returns the canonical created topic for the given entity.
func CreatedTopic(entity string) string {
	return buildEntityTopic(entity, ActionCreated)
}

// UpdatedTopic returns the canonical updated topic for the given entity.
func UpdatedTopic(entity string) string {
	return buildEntityTopic(entity, ActionUpdated)
}

// DeletedTopic returns the canonical deleted topic for the given entity.
func DeletedTopic(entity string) string {
	return buildEntityTopic(entity, ActionDeleted)
}

// CustomTopic returns the canonical topic for the given entity and action.
func CustomTopic(entity, action string) string {
	return buildEntityTopic(entity, action)
}

func buildEntityTopic(entity, action string) string {
	cleanEntity := strings.TrimSpace(entity)
	cleanAction := strings.TrimSpace(action)
	if cleanEntity == "" || cleanAction == "" {
		return ""
	}
	return cleanEntity + "." + cleanAction
}
