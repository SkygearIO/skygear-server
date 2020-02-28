package sso

import (
	"net/url"

	"github.com/skygeario/skygear-server/pkg/core/config"
)

const (
	linkedinAuthorizationURL string = "https://www.linkedin.com/oauth/v2/authorization"
	// nolint: gosec
	linkedinTokenURL    string = "https://www.linkedin.com/oauth/v2/accessToken"
	linkedinUserInfoURL string = "https://www.linkedin.com/v1/people/~?format=json"
)

type LinkedInImpl struct {
	URLPrefix      *url.URL
	OAuthConfig    *config.OAuthConfiguration
	ProviderConfig config.OAuthProviderConfiguration
}

func (f *LinkedInImpl) Type() config.OAuthProviderType {
	return config.OAuthProviderTypeLinkedIn
}

func (f *LinkedInImpl) GetAuthURL(state State, encodedState string) (string, error) {
	p := authURLParams{
		oauthConfig:    f.OAuthConfig,
		urlPrefix:      f.URLPrefix,
		providerConfig: f.ProviderConfig,
		encodedState:   encodedState,
		baseURL:        linkedinAuthorizationURL,
	}
	return authURL(p)
}

func (f *LinkedInImpl) GetAuthInfo(r OAuthAuthorizationResponse, state State) (authInfo AuthInfo, err error) {
	return f.NonOpenIDConnectGetAuthInfo(r, state)
}

func (f *LinkedInImpl) NonOpenIDConnectGetAuthInfo(r OAuthAuthorizationResponse, state State) (authInfo AuthInfo, err error) {
	h := getAuthInfoRequest{
		urlPrefix:      f.URLPrefix,
		oauthConfig:    f.OAuthConfig,
		providerConfig: f.ProviderConfig,
		accessTokenURL: linkedinTokenURL,
		userProfileURL: linkedinUserInfoURL,
		processor:      NewDefaultUserInfoDecoder(),
	}
	return h.getAuthInfo(r, state)
}

func (f *LinkedInImpl) ExternalAccessTokenGetAuthInfo(accessTokenResp AccessTokenResp) (authInfo AuthInfo, err error) {
	h := getAuthInfoRequest{
		urlPrefix:      f.URLPrefix,
		oauthConfig:    f.OAuthConfig,
		providerConfig: f.ProviderConfig,
		accessTokenURL: linkedinTokenURL,
		userProfileURL: linkedinUserInfoURL,
		processor:      NewDefaultUserInfoDecoder(),
	}
	return h.getAuthInfoByAccessTokenResp(accessTokenResp)
}

var (
	_ OAuthProvider                   = &LinkedInImpl{}
	_ NonOpenIDConnectProvider        = &LinkedInImpl{}
	_ ExternalAccessTokenFlowProvider = &LinkedInImpl{}
)
