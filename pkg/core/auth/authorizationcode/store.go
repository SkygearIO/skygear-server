package authorizationcode

import (
	"github.com/skygeario/skygear-server/pkg/core/errors"
)

const Length = 64

var ErrInvalidCode = errors.New("invalid authorization code")

// Store represents the backing store for authorization code.
// Authorization code is cryptographically random string.
type Store interface {
	// New generates the code and stores s.
	New(t *T) (code string, err error)
	// Consumes looks up the code and remove it from the store.
	Consume(code string) (t *T, err error)
}
