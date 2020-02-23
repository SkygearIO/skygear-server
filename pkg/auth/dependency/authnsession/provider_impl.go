package authnsession

import (
	"net/http"
	gotime "time"

	"github.com/dgrijalva/jwt-go"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authnsession"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/event"
	"github.com/skygeario/skygear-server/pkg/core/auth/mfa"
	coreAuthModel "github.com/skygeario/skygear-server/pkg/core/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/model/format"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
	"github.com/skygeario/skygear-server/pkg/core/time"
)

type providerImpl struct {
	authContextGetter                  auth.ContextGetter
	mfaConfiguration                   *config.MFAConfiguration
	authenticationSessionConfiguration *config.AuthenticationSessionConfiguration
	timeProvider                       time.Provider
	mfaProvider                        mfa.Provider
	authInfoStore                      authinfo.Store
	sessionProvider                    session.Provider
	sessionWriter                      session.Writer
	identityProvider                   principal.IdentityProvider
	hookProvider                       hook.Provider
	userProfileStore                   userprofile.Store
}

func NewProvider(
	authContextGetter auth.ContextGetter,
	mfaConfiguration *config.MFAConfiguration,
	authenticationSessionConfiguration *config.AuthenticationSessionConfiguration,
	timeProvider time.Provider,
	mfaProvider mfa.Provider,
	authInfoStore authinfo.Store,
	sessionProvider session.Provider,
	sessionWriter session.Writer,
	identityProvider principal.IdentityProvider,
	hookProvider hook.Provider,
	userProfileStore userprofile.Store,
) Provider {
	return &providerImpl{
		authContextGetter:                  authContextGetter,
		mfaConfiguration:                   mfaConfiguration,
		authenticationSessionConfiguration: authenticationSessionConfiguration,
		timeProvider:                       timeProvider,
		mfaProvider:                        mfaProvider,
		authInfoStore:                      authInfoStore,
		sessionProvider:                    sessionProvider,
		sessionWriter:                      sessionWriter,
		identityProvider:                   identityProvider,
		hookProvider:                       hookProvider,
		userProfileStore:                   userProfileStore,
	}
}

func NewAuthenticationSessionError(token string, step auth.AuthnSessionStep) error {
	return auth.AuthenticationSessionRequired.NewWithInfo(
		"authentication session is required",
		skyerr.Details{"token": token, "step": step},
	)
}

func (p *providerImpl) NewFromToken(token string) (*auth.AuthnSession, error) {
	claims, err := auth.ParseAuthnSessionToken(p.authenticationSessionConfiguration.Secret, token)
	if err != nil {
		return nil, err
	}
	return &claims.AuthnSession, nil
}

func (p *providerImpl) NewFromScratch(userID string, prin principal.Principal, reason auth.SessionCreateReason) (*auth.AuthnSession, error) {
	now := p.timeProvider.NowUTC()
	clientID := p.authContextGetter.AccessKey().ClientID
	requiredSteps, err := authnsession.GetRequiredSteps(p.mfaProvider, p.mfaConfiguration, userID)
	if err != nil {
		return nil, errors.HandledWithMessage(err, "cannot get required authn steps")
	}
	// Identity is considered finished here.
	finishedSteps := requiredSteps[:1]
	return &auth.AuthnSession{
		ClientID:            clientID,
		UserID:              userID,
		PrincipalID:         prin.PrincipalID(),
		PrincipalType:       auth.PrincipalType(prin.ProviderID()),
		PrincipalUpdatedAt:  now,
		RequiredSteps:       requiredSteps,
		FinishedSteps:       finishedSteps,
		SessionCreateReason: reason,
	}, nil
}

func (p *providerImpl) GenerateResponseAndUpdateLastLoginAt(authnSess auth.AuthnSession) (interface{}, error) {
	step, ok := authnSess.NextStep()
	if !ok {
		var authInfo authinfo.AuthInfo
		err := p.authInfoStore.GetAuth(authnSess.UserID, &authInfo)
		if err != nil {
			return nil, err
		}

		userProfile, err := p.userProfileStore.GetUserProfile(authnSess.UserID)
		if err != nil {
			return nil, err
		}

		user := coreAuthModel.NewUser(authInfo, userProfile)

		prin, err := p.identityProvider.GetPrincipalByID(authnSess.PrincipalID)
		if err != nil {
			return nil, err
		}
		identity := coreAuthModel.NewIdentity(p.identityProvider, prin)

		beforeCreate := func(sess *auth.Session) error {
			sessionModel := format.SessionFromSession(sess)
			return p.hookProvider.DispatchEvent(
				event.SessionCreateEvent{
					Reason:   auth.SessionCreateReason(authnSess.SessionCreateReason),
					User:     user,
					Identity: identity,
					Session:  sessionModel,
				},
				&user,
			)
		}
		_, tokens, err := p.sessionProvider.Create(&authnSess, beforeCreate)
		if err != nil {
			return nil, err
		}

		resp := model.NewAuthResponse(user, identity, tokens, authnSess.AuthenticatorBearerToken)

		// Refetch the authInfo
		err = p.authInfoStore.GetAuth(authnSess.UserID, &authInfo)
		if err != nil {
			return nil, err
		}

		// Update LastLoginAt and LastSeenAt
		now := p.timeProvider.NowUTC()
		authInfo.LastLoginAt = &now
		authInfo.LastSeenAt = &now
		authInfo.RefreshDisabledStatus()
		err = p.authInfoStore.UpdateAuth(&authInfo)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}
	now := p.timeProvider.NowUTC()
	expiresAt := now.Add(5 * gotime.Minute)
	claims := auth.AuthnSessionClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
			IssuedAt:  now.Unix(),
		},
		AuthnSession: authnSess,
	}
	token, err := auth.NewAuthnSessionToken(p.authenticationSessionConfiguration.Secret, claims)
	if err != nil {
		return nil, err
	}
	authnSessionErr := NewAuthenticationSessionError(token, step)
	return authnSessionErr, nil
}

func (p *providerImpl) GenerateResponseWithSession(sess *auth.Session, mfaBearerToken string) (interface{}, error) {
	var authInfo authinfo.AuthInfo
	err := p.authInfoStore.GetAuth(sess.UserID, &authInfo)
	if err != nil {
		return nil, err
	}

	userProfile, err := p.userProfileStore.GetUserProfile(sess.UserID)
	if err != nil {
		return nil, err
	}
	user := coreAuthModel.NewUser(authInfo, userProfile)

	prin, err := p.identityProvider.GetPrincipalByID(sess.PrincipalID)
	if err != nil {
		return nil, err
	}
	identity := coreAuthModel.NewIdentity(p.identityProvider, prin)

	resp := model.NewAuthResponse(user, identity, auth.SessionTokens{ID: sess.ID}, mfaBearerToken)
	return resp, nil
}

func (p *providerImpl) WriteResponse(w http.ResponseWriter, resp interface{}, err error) {
	if err == nil {
		switch v := resp.(type) {
		case model.AuthResponse:
			// Do not touch the cookie if it is not in the response.
			if v.MFABearerToken == "" {
				p.sessionWriter.WriteSession(w, &v.AccessToken, nil)
			} else {
				p.sessionWriter.WriteSession(w, &v.AccessToken, &v.MFABearerToken)
			}
			handler.WriteResponse(w, handler.APIResponse{Result: v})
		case error:
			handler.WriteResponse(w, handler.APIResponse{Error: v})
		default:
			panic("authnsession: unknown response")
		}
	} else {
		if skyerr.IsKind(err, mfa.InvalidBearerToken) {
			p.sessionWriter.ClearMFABearerToken(w)
		}
		handler.WriteResponse(w, handler.APIResponse{Error: err})
	}
}

func (p *providerImpl) Resolve(authContext auth.ContextGetter, authnSessionToken string, options ResolveOptions) (userID string, sess *auth.Session, authnSession *auth.AuthnSession, err error) {
	// Simple case
	sess, _ = authContext.Session()
	if sess != nil {
		userID = sess.UserID
		return
	}

	if authnSessionToken == "" {
		err = authz.ErrNotAuthenticated
		return
	}

	authnSession, err = p.NewFromToken(authnSessionToken)
	if err != nil {
		return
	}

	step, ok := authnSession.NextStep()
	if !ok {
		err = auth.ErrInvalidAuthnSessionToken
		return
	}

	switch step {
	case auth.AuthnSessionStepMFA:
		switch options.MFAOption {
		case ResolveMFAOptionAlwaysAccept:
			userID = authnSession.UserID
			return
		case ResolveMFAOptionOnlyWhenNoAuthenticators:
			var authenticators []mfa.Authenticator
			authenticators, err = p.mfaProvider.ListAuthenticators(authnSession.UserID)
			if err != nil {
				return
			}
			if len(authenticators) > 0 {
				err = authz.ErrNotAuthenticated
				return
			}
			userID = authnSession.UserID
			return
		}
	}

	err = errors.New("unexpected authn session state")
	return
}
