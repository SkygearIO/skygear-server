package customtoken

import (
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/uuid"
)

type Principal struct {
	ID               string
	UserID           string
	TokenPrincipalID string
	RawProfile       SSOCustomTokenClaims
	ClaimsValue      map[string]interface{}
}

func NewPrincipal() Principal {
	return Principal{
		ID: uuid.New(),
	}
}

func (p *Principal) PrincipalID() string {
	return p.ID
}

func (p *Principal) PrincipalUserID() string {
	return p.UserID
}

func (p *Principal) ProviderID() string {
	return string(coreAuth.PrincipalTypeCustomToken)
}

func (p *Principal) Attributes() principal.Attributes {
	return principal.Attributes{
		"provider_user_id": p.TokenPrincipalID,
		"raw_profile":      p.RawProfile,
	}
}

func (p *Principal) Claims() principal.Claims {
	return p.ClaimsValue
}

func (p *Principal) SetRawProfile(rawProfile SSOCustomTokenClaims) {
	p.RawProfile = rawProfile

	claimsValue := map[string]interface{}{}
	email := rawProfile.Email()
	if email != "" {
		claimsValue["email"] = email
	}
	p.ClaimsValue = claimsValue
}
