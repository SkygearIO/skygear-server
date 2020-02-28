package event

import (
	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/model"
)

const (
	BeforeSessionCreate Type = "before_session_create"
	AfterSessionCreate  Type = "after_session_create"
)

/*
	@Callback
		@Operation POST /before_session_create - Before session creation
			A session is about to be created.
			@RequestBody
				@JSONSchema {BeforeSessionCreateEvent}
			@Response 200 {HookResponse}

		@Operation POST /after_session_create - After session creation
			A session is created.
			@RequestBody
				@JSONSchema {AfterSessionCreateEvent}
			@Response 200 {EmptyResponse}
*/
type SessionCreateEvent struct {
	Reason   auth.SessionCreateReason `json:"reason"`
	User     model.User               `json:"user"`
	Identity model.Identity           `json:"identity"`
	Session  model.Session            `json:"session"`
}

// @JSONSchema
const BeforeSessionCreateEventSchema = `
{
	"$id": "#BeforeSessionCreateEvent",
	"type": "object",
	"properties": {
		"id": { "type": "string" },
		"seq": { "type": "integer" },
		"type": { "type": "string", "enum": ["before_session_create"] },
		"payload": { "$ref": "#SessionCreateEventPayload" },
		"context": { "$ref": "#EventContext" }
	}
}
`

// @JSONSchema
const AfterSessionCreateEventSchema = `
{
	"$id": "#AfterSessionCreateEvent",
	"type": "object",
	"properties": {
		"id": { "type": "string" },
		"seq": { "type": "integer" },
		"type": { "type": "string", "enum": ["after_session_create"] },
		"payload": { "$ref": "#SessionCreateEventPayload" },
		"context": { "$ref": "#EventContext" }
	}
}
`

// @JSONSchema
const SessionCreateEventPayloadSchema = `
{
	"$id": "#SessionCreateEventPayload",
	"type": "object",
	"properties": {
		"reason": { "type": "string" },
		"user": { "$ref": "#User" },
		"identity": { "$ref": "#Identity" },
		"session": { "$ref": "#Session" }
	}
}
`

func (SessionCreateEvent) BeforeEventType() Type {
	return BeforeSessionCreate
}

func (SessionCreateEvent) AfterEventType() Type {
	return AfterSessionCreate
}

func (event SessionCreateEvent) WithMutationsApplied(mutations Mutations) UserAwarePayload {
	user := event.User
	mutations.ApplyToUser(&user)
	return SessionCreateEvent{
		Reason:   event.Reason,
		User:     user,
		Identity: event.Identity,
	}
}

func (event SessionCreateEvent) UserID() string {
	return event.User.ID
}
