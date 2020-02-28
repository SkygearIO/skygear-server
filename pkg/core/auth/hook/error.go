package hook

import (
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
)

var WebHookDisallowed = skyerr.Forbidden.WithReason("WebHookDisallowed")

var errDeliveryTimeout = errors.New("web-hook event delivery timed out")
var errDeliveryInvalidStatusCode = errors.New("invalid status code")

func newErrorDeliveryFailed(inner error) error {
	return errors.Newf("web-hook event delivery failed: %w", inner)
}

type OperationDisallowedItem struct {
	Reason string      `json:"reason"`
	Data   interface{} `json:"data,omitempty"`
}

func newErrorOperationDisallowed(items []OperationDisallowedItem) error {
	// NOTE(error): These are not causes. Causes are pre-defined,
	// and reasons are provided by hook handlers.
	return WebHookDisallowed.NewWithInfo(
		"disallowed by web-hook event handler",
		map[string]interface{}{"reasons": items},
	)
}

func newErrorMutationFailed(inner error) error {
	return errors.Newf("web-hook mutation failed: %w", inner)
}
