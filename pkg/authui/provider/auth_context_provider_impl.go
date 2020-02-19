package provider

import (
	"crypto/subtle"
	"net/http"

	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/model"
)

type AuthContextProviderImpl struct {
	APIClients []config.APIClientConfiguration

	accessKey model.AccessKey
	authInfo  *authinfo.AuthInfo
	session   *coreAuth.Session
	err       error
}

var _ AuthContextProvider = &AuthContextProviderImpl{}

func NewAuthContextProvider(tConfig *config.TenantConfiguration, r *http.Request) *AuthContextProviderImpl {
	return &AuthContextProviderImpl{
		APIClients: tConfig.AppConfig.Clients,
		accessKey: model.AccessKey{
			Type: model.NoAccessKeyType,
		},
	}
}

func (p *AuthContextProviderImpl) Init(r *http.Request) {
	// Assume r.Form is parsed.
	clientID := r.Form.Get("client_id")
	name := ""
	for _, clientConfig := range p.APIClients {
		if subtle.ConstantTimeCompare([]byte(clientID), []byte(clientConfig.APIKey)) == 1 {
			name = clientConfig.ID
		}
	}
	if name != "" {
		p.accessKey.Type = model.APIAccessKeyType
		p.accessKey.ClientID = name
	}

	// TODO(authui): set authInfo
	// TODO(authui): set session
}

func (p *AuthContextProviderImpl) AccessKey() model.AccessKey {
	return p.accessKey
}

func (p *AuthContextProviderImpl) AuthInfo() (*authinfo.AuthInfo, error) {
	return p.authInfo, p.err
}

func (p *AuthContextProviderImpl) Session() (*coreAuth.Session, error) {
	return p.session, p.err
}
