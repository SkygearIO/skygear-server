package authorizationcode

import (
	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/errors"
)

const Length = 64

var ErrInvalidCode = errors.New("invalid authorization code")

// Store represents the backing store for authorization code.
// Authorization code is cryptographically random string.
// It is used to retrieve a AuthnSession.
type Store interface {
	// New generates the code and stores s.
	New(s *auth.AuthnSession) (code string, err error)
	// Consumes looks up the code and remove it from the store.
	Consume(code string) (s *auth.AuthnSession, err error)
}
