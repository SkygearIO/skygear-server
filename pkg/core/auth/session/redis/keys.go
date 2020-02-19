package redis

import "fmt"

type SessionKeyFunc func(appID string, sessionID string) string

type SessionListKeyFunc func(appID string, userID string) string

type EventStreamKeyFunc func(appID string, sessionID string) string

var SessionKey = SessionKeyFunc(func(appID string, sessionID string) string {
	return fmt.Sprintf("%s:session:%s", appID, sessionID)
})

var SessionListKey = SessionListKeyFunc(func(appID string, userID string) string {
	return fmt.Sprintf("%s:session-list:%s", appID, userID)
})

var EventStreamKey = EventStreamKeyFunc(func(appID string, sessionID string) string {
	return fmt.Sprintf("%s:event:%s", appID, sessionID)
})
