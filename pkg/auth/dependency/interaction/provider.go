package interaction

import (
	gotime "time"

	"github.com/sirupsen/logrus"
	"github.com/skygeario/skygear-server/pkg/core/authn"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/time"
)

type IdentityProvider interface {
	Get(userID string, typ authn.IdentityType, id string) (*IdentityInfo, error)
	GetByClaims(typ authn.IdentityType, claims map[string]interface{}) (string, *IdentityInfo, error)
	New(userID string, typ authn.IdentityType, claims map[string]interface{}) *IdentityInfo
}

type AuthenticatorProvider interface {
	Get(userID string, typ AuthenticatorType, id string) (*AuthenticatorInfo, error)
	List(userID string, typ AuthenticatorType) ([]*AuthenticatorInfo, error)
	New(userID string, spec AuthenticatorSpec, secret string) ([]*AuthenticatorInfo, error)
	Authenticate(userID string, spec AuthenticatorSpec, state *map[string]string, secret string) (*AuthenticatorInfo, error)
}

type UserProvider interface {
	Create(userID string, metadata map[string]interface{}, identities []*IdentityInfo) error
}

// TODO(interaction): configurable lifetime
const interactionIdleTimeout = 5 * gotime.Minute

type Provider struct {
	Store         Store
	Time          time.Provider
	Logger        *logrus.Entry
	Identity      IdentityProvider
	Authenticator AuthenticatorProvider
	User          UserProvider
	Config        *config.AuthenticationConfiguration
}

func (p *Provider) GetInteraction(token string) (*Interaction, error) {
	i, err := p.Store.Get(token)
	if err != nil {
		return nil, err
	}

	if i.Identity != nil && !i.IsNewIdentity(i.Identity.ID) {
		if i.Identity, err = p.Identity.Get(
			i.UserID, i.Identity.Type, i.Identity.ID); err != nil {
			return nil, err
		}
	}
	if i.PrimaryAuthenticator != nil && !i.IsNewAuthenticator(i.PrimaryAuthenticator.ID) {
		if i.PrimaryAuthenticator, err = p.Authenticator.Get(
			i.UserID, i.PrimaryAuthenticator.Type, i.PrimaryAuthenticator.ID); err != nil {
			return nil, err
		}
	}
	if i.SecondaryAuthenticator != nil && !i.IsNewAuthenticator(i.SecondaryAuthenticator.ID) {
		if i.SecondaryAuthenticator, err = p.Authenticator.Get(
			i.UserID, i.SecondaryAuthenticator.Type, i.SecondaryAuthenticator.ID); err != nil {
			return nil, err
		}
	}

	return i, nil
}

func (p *Provider) SaveInteraction(i *Interaction) (string, error) {
	if i.Token == "" {
		i.Token = generateToken()
		i.CreatedAt = p.Time.NowUTC()
		i.ExpireAt = i.CreatedAt.Add(interactionIdleTimeout)
		if err := p.Store.Create(i); err != nil {
			return "", err
		}
	} else {
		i.ExpireAt = p.Time.NowUTC().Add(interactionIdleTimeout)
		if err := p.Store.Update(i); err != nil {
			return "", err
		}
	}
	return i.Token, nil
}
