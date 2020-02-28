package sso

import (
	"crypto/subtle"
	"net/http"
	"net/url"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/crypto"
	coreTime "github.com/skygeario/skygear-server/pkg/core/time"
)

const (
	googleOIDCDiscoveryDocumentURL string = "https://accounts.google.com/.well-known/openid-configuration"
	// nolint: gosec
	googleTokenURL    string = "https://www.googleapis.com/oauth2/v4/token"
	googleUserInfoURL string = "https://www.googleapis.com/oauth2/v1/userinfo"
)

type GoogleImpl struct {
	URLPrefix      *url.URL
	OAuthConfig    *config.OAuthConfiguration
	ProviderConfig config.OAuthProviderConfiguration
	TimeProvider   coreTime.Provider
}

func (f *GoogleImpl) GetAuthURL(state State, encodedState string) (string, error) {
	d, err := FetchOIDCDiscoveryDocument(http.DefaultClient, googleOIDCDiscoveryDocumentURL)
	if err != nil {
		return "", err
	}
	return d.MakeOAuthURL(OIDCAuthParams{
		ProviderConfig: f.ProviderConfig,
		URLPrefix:      f.URLPrefix,
		Nonce:          state.Nonce,
		EncodedState:   encodedState,
		ExtraParams: map[string]string{
			"prompt": "select_account",
		},
	}), nil
}

func (f *GoogleImpl) Type() config.OAuthProviderType {
	return config.OAuthProviderTypeGoogle
}

func (f *GoogleImpl) GetAuthInfo(r OAuthAuthorizationResponse, state State) (authInfo AuthInfo, err error) {
	return f.OpenIDConnectGetAuthInfo(r, state)
}

func (f *GoogleImpl) OpenIDConnectGetAuthInfo(r OAuthAuthorizationResponse, state State) (authInfo AuthInfo, err error) {
	if subtle.ConstantTimeCompare([]byte(state.Nonce), []byte(crypto.SHA256String(r.Nonce))) != 1 {
		err = NewSSOFailed(InvalidParams, "invalid sso state")
		return
	}

	d, err := FetchOIDCDiscoveryDocument(http.DefaultClient, googleOIDCDiscoveryDocumentURL)
	if err != nil {
		err = NewSSOFailed(NetworkFailed, "failed to get OIDC discovery document")
		return
	}
	// TODO(sso): Cache JWKs
	keySet, err := d.FetchJWKs(http.DefaultClient)
	if err != nil {
		err = NewSSOFailed(NetworkFailed, "failed to get OIDC JWKs")
		return
	}

	var tokenResp AccessTokenResp
	claims, err := d.ExchangeCode(
		http.DefaultClient,
		r.Code,
		keySet,
		f.URLPrefix,
		f.ProviderConfig.ClientID,
		f.ProviderConfig.ClientSecret,
		redirectURI(f.URLPrefix, f.ProviderConfig),
		r.Nonce,
		f.TimeProvider.NowUTC,
		&tokenResp,
	)
	if err != nil {
		return
	}

	// Verify the issuer
	// https://developers.google.com/identity/protocols/OpenIDConnect#validatinganidtoken
	iss, ok := claims["iss"].(string)
	if !ok {
		err = NewSSOFailed(SSOUnauthorized, "invalid iss")
		return
	}
	if iss != "https://accounts.google.com" && iss != "accounts.google.com" {
		err = NewSSOFailed(SSOUnauthorized, "invalid iss")
		return
	}

	// Ensure sub exists
	sub, ok := claims["sub"].(string)
	if !ok {
		err = NewSSOFailed(SSOUnauthorized, "no sub")
		return
	}

	email, _ := claims["email"].(string)

	authInfo.ProviderConfig = f.ProviderConfig
	authInfo.ProviderRawProfile = claims
	authInfo.ProviderAccessTokenResp = tokenResp
	authInfo.ProviderUserInfo = ProviderUserInfo{
		ID:    sub,
		Email: email,
	}

	return
}

func (f *GoogleImpl) ExternalAccessTokenGetAuthInfo(accessTokenResp AccessTokenResp) (authInfo AuthInfo, err error) {
	h := getAuthInfoRequest{
		urlPrefix:      f.URLPrefix,
		oauthConfig:    f.OAuthConfig,
		providerConfig: f.ProviderConfig,
		accessTokenURL: googleTokenURL,
		userProfileURL: googleUserInfoURL,
		processor:      NewDefaultUserInfoDecoder(),
	}
	return h.getAuthInfoByAccessTokenResp(accessTokenResp)
}

var (
	_ OAuthProvider                   = &GoogleImpl{}
	_ OpenIDConnectProvider           = &GoogleImpl{}
	_ ExternalAccessTokenFlowProvider = &GoogleImpl{}
)
