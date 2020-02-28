package hook

import (
	"github.com/skygeario/skygear-server/pkg/core/auth/event"
	"github.com/skygeario/skygear-server/pkg/core/auth/model"
)

type MockProvider struct {
	DispatchedEvents []event.Payload
}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (provider *MockProvider) DispatchEvent(payload event.Payload, user *model.User) error {
	provider.DispatchedEvents = append(provider.DispatchedEvents, payload)
	return nil
}

func (MockProvider) WillCommitTx() error {
	return nil
}

func (MockProvider) DidCommitTx() {

}

var _ Provider = &MockProvider{}
