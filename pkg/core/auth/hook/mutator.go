package hook

import (
	"github.com/skygeario/skygear-server/pkg/core/auth/event"
	"github.com/skygeario/skygear-server/pkg/core/auth/model"
)

type Mutator interface {
	New(event *event.Event, user *model.User) Mutator
	Add(event.Mutations) error
	Apply() error
}
