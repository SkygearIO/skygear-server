package provider

import (
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/sirupsen/logrus"

	"github.com/skygeario/skygear-server/pkg/core/audit"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authnsession"
	"github.com/skygeario/skygear-server/pkg/core/auth/mfa"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal/password"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	coreTime "github.com/skygeario/skygear-server/pkg/core/time"
)

type AuthenticationProviderImpl struct {
	PassworAuthProvider                password.Provider
	AuditTrail                         audit.Trail
	Logger                             *logrus.Entry
	TimeProvider                       coreTime.Provider
	AuthContextProvider                AuthContextProvider
	MFAProvider                        mfa.Provider
	MFAConfiguration                   *config.MFAConfiguration
	AuthenticationSessionConfiguration *config.AuthenticationSessionConfiguration
}

var _ AuthenticationProvider = &AuthenticationProviderImpl{}

func (p *AuthenticationProviderImpl) FromToken(token string) (*coreAuth.AuthnSession, error) {
	claims, err := coreAuth.ParseAuthnSessionToken(p.AuthenticationSessionConfiguration.Secret, token)
	if err != nil {
		return nil, err
	}
	return &claims.AuthnSession, nil
}

func (p *AuthenticationProviderImpl) ToToken(authnSession *coreAuth.AuthnSession) (token string, err error) {
	now := p.TimeProvider.NowUTC()
	expiresAt := now.Add(5 * time.Minute)
	claims := coreAuth.AuthnSessionClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
			IssuedAt:  now.Unix(),
		},
		AuthnSession: *authnSession,
	}
	token, err = coreAuth.NewAuthnSessionToken(p.AuthenticationSessionConfiguration.Secret, claims)
	if err != nil {
		return
	}
	return
}

func (p *AuthenticationProviderImpl) AuthenticateWithPassword(loginID string, plainPassword string) (authnSession *coreAuth.AuthnSession, err error) {
	var userID string

	defer func() {
		if userID != "" {
			if err != nil {
				p.AuditTrail.Log(audit.Entry{
					UserID: userID,
					Event:  audit.EventLoginFailure,
				})
			} else {
				p.AuditTrail.Log(audit.Entry{
					UserID: userID,
					Event:  audit.EventLoginSuccess,
				})
			}
		}
	}()

	var prin password.Principal
	err = p.PassworAuthProvider.GetPrincipalByLoginIDWithRealm("", loginID, "", &prin)
	if err != nil {
		if errors.Is(err, principal.ErrNotFound) {
			err = password.ErrInvalidCredentials
		}
		if errors.Is(err, principal.ErrMultipleResultsFound) {
			p.Logger.WithError(err).Warn("Multiple results found for password principal query")
			err = password.ErrInvalidCredentials
		}
		return
	}
	userID = prin.UserID

	err = prin.VerifyPassword(plainPassword)
	if err != nil {
		return
	}

	// This err is non-critical
	if err := p.PassworAuthProvider.MigratePassword(&prin, plainPassword); err != nil {
		p.Logger.WithError(err).Error("Failed to migrate password")
	}

	now := p.TimeProvider.NowUTC()
	requiredSteps, err := authnsession.GetRequiredSteps(p.MFAProvider, p.MFAConfiguration, userID)
	if err != nil {
		err = errors.HandledWithMessage(err, "cannot get required authn steps")
		return
	}

	// Identity is considered finished here.
	finishedSteps := requiredSteps[:1]
	authnSession = &coreAuth.AuthnSession{
		// No client ID
		ClientID:            "",
		UserID:              userID,
		PrincipalID:         prin.PrincipalID(),
		PrincipalType:       coreAuth.PrincipalType(prin.ProviderID()),
		PrincipalUpdatedAt:  now,
		RequiredSteps:       requiredSteps,
		FinishedSteps:       finishedSteps,
		SessionCreateReason: coreAuth.SessionCreateReasonLogin,
	}

	return
}
