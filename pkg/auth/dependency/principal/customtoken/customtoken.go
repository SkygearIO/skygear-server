package customtoken

import (
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
)

type Provider interface {
	principal.Provider
	Decode(tokenString string) (SSOCustomTokenClaims, error)
	CreatePrincipal(principal *Principal) error
	UpdatePrincipal(principal *Principal) error
	GetPrincipalByTokenPrincipalID(tokenPrincipalID string) (*Principal, error)
}
