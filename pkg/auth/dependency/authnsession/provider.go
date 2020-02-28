package authnsession

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth"
)

type ResolveOptions struct {
	MFAOption ResolveMFAOption
}

type ResolveMFAOption int

const (
	ResolveMFAOptionAlwaysAccept ResolveMFAOption = iota
	ResolveMFAOptionOnlyWhenNoAuthenticators
)

// Provider manipulates authentication session
type Provider interface {
	// NewFromToken decodes an authentication session from a token.
	NewFromToken(token string) (*auth.AuthnSession, error)
	// NewFromScratch creates a new authentication session.
	NewFromScratch(userID string, prin principal.Principal, reason auth.SessionCreateReason) (*auth.AuthnSession, error)
	// GenerateResponseAndUpdateLastLoginAt generates authentication response and update last_login_at
	// if the response is AuthResponse.

	GenerateResponseAndUpdateLastLoginAt(session auth.AuthnSession) (interface{}, error)

	// GenerateResponseWithSession generates authentication response.
	GenerateResponseWithSession(sess *auth.Session, mfaBearerToken string) (interface{}, error)

	// WriteResponse alters the response, write Cookies and write HTTP Body. It should be used in a defer block.
	// It should be used in most cases.
	WriteResponse(w http.ResponseWriter, resp interface{}, err error)

	// Resolve resolves session or authentication session.
	Resolve(authContext auth.ContextGetter, authnSessionToken string, options ResolveOptions) (userID string, sess *auth.Session, authnSession *auth.AuthnSession, err error)
}
