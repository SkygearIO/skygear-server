package redis

import (
	"context"
	"encoding/json"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	"github.com/skygeario/skygear-server/pkg/core/redis"
)

// TODO(session): tune event persistence, maybe use other datastore
const maxEventStreamLength = 1000

const eventTypeAccessEvent = "access"

type EventStoreImpl struct {
	ctx   context.Context
	appID string
}

var _ session.EventStore = &EventStoreImpl{}

func NewEventStore(ctx context.Context, appID string) *EventStoreImpl {
	return &EventStoreImpl{ctx: ctx, appID: appID}
}

func (s *EventStoreImpl) AppendAccessEvent(session *auth.Session, event *auth.SessionAccessEvent) (err error) {
	json, err := json.Marshal(event)
	if err != nil {
		return
	}

	conn := redis.GetConn(s.ctx)
	key := eventStreamKey(s.appID, session.ID)

	args := []interface{}{key}
	if maxEventStreamLength >= 0 {
		args = append(args, "MAXLEN", "~", maxEventStreamLength)
	}
	args = append(args, "*", eventTypeAccessEvent, json)

	_, err = conn.Do("XADD", args...)
	return
}
