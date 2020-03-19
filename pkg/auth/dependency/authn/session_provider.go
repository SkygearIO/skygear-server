package authn

import (
	gotime "time"

	"github.com/dgrijalva/jwt-go"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/mfa"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/session"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/auth/event"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/authn"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	coremodel "github.com/skygeario/skygear-server/pkg/core/model"
	"github.com/skygeario/skygear-server/pkg/core/time"
)

type SessionProvider struct {
	MFAProvider        mfa.Provider
	SessionProvider    session.Provider
	ClientConfigs      []config.OAuthClientConfiguration
	MFAConfig          *config.MFAConfiguration
	AuthnSessionConfig *config.AuthenticationSessionConfiguration
	TimeProvider       time.Provider
	AuthInfoStore      authinfo.Store
	UserProfileStore   userprofile.Store
	IdentityProvider   principal.IdentityProvider
	HookProvider       hook.Provider
}

func (p *SessionProvider) BeginSession(client config.OAuthClientConfiguration, userID string, prin principal.Principal, reason session.CreateReason) (*AuthnSession, error) {
	now := p.TimeProvider.NowUTC()
	requiredSteps, err := p.getRequiredSteps(userID)
	if err != nil {
		return nil, errors.HandledWithMessage(err, "cannot get required authn steps")
	}
	// Identity is considered finished here.
	finishedSteps := requiredSteps[:1]
	clientID := ""
	if client != nil {
		clientID = client.ClientID()
	}
	return &AuthnSession{
		ClientID: clientID,
		Attrs: authn.Attrs{
			UserID:             userID,
			PrincipalID:        prin.PrincipalID(),
			PrincipalType:      authn.PrincipalType(prin.ProviderID()),
			PrincipalUpdatedAt: now,
		},
		RequiredSteps:       requiredSteps,
		FinishedSteps:       finishedSteps,
		SessionCreateReason: reason,
	}, nil
}

func (p *SessionProvider) StepSession(s *AuthnSession) (Result, error) {
	var client config.OAuthClientConfiguration
	var ok bool
	if s.ClientID != "" {
		client, ok = coremodel.GetClientConfig(p.ClientConfigs, s.ClientID)
		if !ok {
			return nil, ErrInvalidAuthenticationSession
		}
	}

	// Step through all finished steps
	step, inProgress := s.NextStep()
	for inProgress && p.isStepFinished(step, &s.Attrs) {
		s.FinishedSteps = append(s.FinishedSteps, step)
		step, inProgress = s.NextStep()
	}

	if !inProgress {
		return p.completeSession(s, client)
	}
	return p.saveSession(s)
}

func (p *SessionProvider) MakeResult(client config.OAuthClientConfiguration, s auth.AuthSession, bearerToken string) (Result, error) {
	user, identity, err := p.loadData(s.AuthnAttrs())
	if err != nil {
		return nil, err
	}
	sessionModel := s.ToAPIModel()

	return &CompletionResult{
		Client:    client,
		User:      user,
		Principal: identity,
		Session:   sessionModel,

		MFABearerToken: bearerToken,
	}, nil
}

func (p *SessionProvider) ResolveSession(jwt string) (*AuthnSession, error) {
	if jwt == "" {
		return nil, authz.ErrNotAuthenticated
	}

	token, err := decodeSessionToken(p.AuthnSessionConfig.Secret, jwt)
	if err != nil {
		return nil, err
	}
	s := token.AuthnSession

	return &s, nil
}

func (p *SessionProvider) loadData(attrs *authn.Attrs) (*model.User, *model.Identity, error) {
	var authInfo authinfo.AuthInfo
	err := p.AuthInfoStore.GetAuth(attrs.UserID, &authInfo)
	if err != nil {
		return nil, nil, err
	}

	userProfile, err := p.UserProfileStore.GetUserProfile(attrs.UserID)
	if err != nil {
		return nil, nil, err
	}

	prin, err := p.IdentityProvider.GetPrincipalByID(attrs.PrincipalID)
	if err != nil {
		return nil, nil, err
	}

	user := model.NewUser(authInfo, userProfile)
	identity := model.NewIdentity(nil, prin)

	return &user, &identity, nil
}

func (p *SessionProvider) completeSession(s *AuthnSession, client config.OAuthClientConfiguration) (Result, error) {
	user, identity, err := p.loadData(&s.Attrs)
	if err != nil {
		return nil, err
	}

	session, token := p.SessionProvider.MakeSession(&s.Attrs)

	sessionModel := session.ToAPIModel()
	err = p.HookProvider.DispatchEvent(
		event.SessionCreateEvent{
			Reason:   s.SessionCreateReason,
			User:     *user,
			Identity: *identity,
			Session:  *sessionModel,
		},
		user,
	)
	if err != nil {
		return nil, err
	}

	err = p.updateLoginTime(session.Attrs.UserID)
	if err != nil {
		return nil, err
	}

	err = p.SessionProvider.Create(session)
	if err != nil {
		return nil, err
	}

	return &CompletionResult{
		Client:    client,
		User:      user,
		Principal: identity,
		Session:   sessionModel,

		SessionToken:   token,
		MFABearerToken: s.AuthenticatorBearerToken,
	}, nil
}

func (p *SessionProvider) saveSession(s *AuthnSession) (Result, error) {
	step, ok := s.NextStep()
	if !ok {
		panic("authn: attempt to save a completed session")
	}

	now := p.TimeProvider.NowUTC()
	// TODO(authn): adjustable expiry
	expiresAt := now.Add(5 * gotime.Minute)
	token := sessionToken{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiresAt.Unix(),
			IssuedAt:  now.Unix(),
		},
		AuthnSession: *s,
	}
	jwt, err := encodeSessionToken(p.AuthnSessionConfig.Secret, token)
	if err != nil {
		return nil, err
	}

	return &InProgressResult{
		AuthnSessionToken: jwt,
		CurrentStep:       step,
	}, nil
}

func (p *SessionProvider) updateLoginTime(userID string) error {
	authInfo := &authinfo.AuthInfo{}
	err := p.AuthInfoStore.GetAuth(userID, authInfo)
	if err != nil {
		return err
	}

	// Update LastLoginAt and LastSeenAt
	now := p.TimeProvider.NowUTC()
	authInfo.LastLoginAt = &now
	authInfo.LastSeenAt = &now
	authInfo.RefreshDisabledStatus(now)
	err = p.AuthInfoStore.UpdateAuth(authInfo)
	if err != nil {
		return err
	}

	return nil
}

func (p *SessionProvider) getRequiredSteps(userID string) ([]SessionStep, error) {
	steps := []SessionStep{SessionStepIdentity}

	// MFA
	enforcement := p.MFAConfig.Enforcement
	if enforcement != config.MFAEnforcementOff {
		authenticators, err := p.MFAProvider.ListAuthenticators(userID)
		if err != nil {
			return nil, err
		}
		if len(authenticators) > 0 {
			// When there are MFA authenticators:
			// perform MFA authn if not turned off.
			steps = append(steps, SessionStepMFAAuthn)
		} else if enforcement == config.MFAEnforcementRequired {
			// When there are no MFA authenticator, and MFA is required:
			// require setup MFA authenticators
			steps = append(steps, SessionStepMFASetup)
		}
	}

	return steps, nil
}

func (p *SessionProvider) isStepFinished(step SessionStep, attrs *authn.Attrs) bool {
	switch step {
	case SessionStepIdentity:
		return attrs.PrincipalID != ""
	case SessionStepMFAAuthn, SessionStepMFASetup:
		return attrs.AuthenticatorID != ""
	default:
		panic("authn: unknown authn session step " + step)
	}
}
