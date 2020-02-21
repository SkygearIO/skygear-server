package auth

import (
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
)

var (
	InvalidAuthenticationSession  skyerr.Kind = skyerr.Invalid.WithReason("InvalidAuthenticationSession")
	AuthenticationSessionRequired skyerr.Kind = skyerr.Unauthorized.WithReason("AuthenticationSession")
)

var ErrInvalidAuthnSessionToken = InvalidAuthenticationSession.New("invalid authentication session token")

type AuthnSessionStep string

const (
	AuthnSessionStepIdentity AuthnSessionStep = "identity"
	AuthnSessionStepMFA      AuthnSessionStep = "mfa"
)

// AuthnSession represents the authentication session.
// When the authentication session is finished, it converts to Session.
type AuthnSession struct {
	// The following fields are filled in step "identity"
	ClientID string `json:"client_id"`
	UserID   string `json:"user_id"`

	PrincipalID        string        `json:"principal_id"`
	PrincipalType      PrincipalType `json:"principal_type"`
	PrincipalUpdatedAt time.Time     `json:"principal_updated_at"`

	RequiredSteps       []AuthnSessionStep  `json:"required_steps"`
	FinishedSteps       []AuthnSessionStep  `json:"finished_steps"`
	SessionCreateReason SessionCreateReason `json:"session_create_reason"`

	// The following fields are filled in step "mfa"
	AuthenticatorID         string                  `json:"authenticator_id,omitempty"`
	AuthenticatorType       AuthenticatorType       `json:"authenticator_type,omitempty"`
	AuthenticatorOOBChannel AuthenticatorOOBChannel `json:"authenticator_oob_channel,omitempty"`
	AuthenticatorUpdatedAt  *time.Time              `json:"authenticator_updated_at,omitempty"`

	AuthenticatorBearerToken string `json:"authenticator_bearer_token,omitempty"`
}

type AuthnSessionStepMFAOptions struct {
	AuthenticatorID          string
	AuthenticatorType        AuthenticatorType
	AuthenticatorOOBChannel  AuthenticatorOOBChannel
	AuthenticatorBearerToken string
}

func (a *AuthnSession) IsFinished() bool {
	return len(a.RequiredSteps) == len(a.FinishedSteps)
}

func (a *AuthnSession) NextStep() (AuthnSessionStep, bool) {
	if a.IsFinished() {
		return "", false
	}
	return a.RequiredSteps[len(a.FinishedSteps)], true
}

type AuthnSessionClaims struct {
	jwt.StandardClaims
	AuthnSession AuthnSession `json:"authn_session"`
}

func NewAuthnSessionToken(secret string, claims AuthnSessionClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseAuthnSessionToken(secret string, tokenString string) (*AuthnSessionClaims, error) {
	t, err := jwt.ParseWithClaims(tokenString, &AuthnSessionClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected JWT alg")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, ErrInvalidAuthnSessionToken
	}
	claims, ok := t.Claims.(*AuthnSessionClaims)
	if !ok {
		return nil, ErrInvalidAuthnSessionToken
	}
	if !t.Valid {
		return nil, ErrInvalidAuthnSessionToken
	}
	return claims, nil
}
