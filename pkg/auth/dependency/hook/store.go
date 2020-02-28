package hook

import "github.com/skygeario/skygear-server/pkg/core/auth/event"

type Store interface {
	NextSequenceNumber() (int64, error)
	AddEvents(events []*event.Event) error
	GetEventsForDelivery() ([]*event.Event, error)
}
