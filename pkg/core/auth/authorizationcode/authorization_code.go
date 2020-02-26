package authorizationcode

import (
	"net/url"

	"github.com/skygeario/skygear-server/pkg/core/auth"
)

type T struct {
	Form         url.Values         `json:"form"`
	AuthnSession *auth.AuthnSession `json:"authn_session"`
}

var authenticationRequestParams = []string{
	// https://tools.ietf.org/html/rfc6749#section-4.1.1
	"response_type",
	"client_id",
	"redirect_uri",
	"scope",
	"state",
	// https://openid.net/specs/openid-connect-core-1_0.html#AuthRequest
	"response_mode",
	"nonce",
	"display",
	"prompt",
	"max_age",
	"ui_locales",
	"id_token_hint",
	"login_hint",
	"acr_values",
	// https://openid.net/specs/openid-connect-core-1_0.html#ClaimsLanguagesAndScripts
	"claims_locales",
	// https://openid.net/specs/openid-connect-core-1_0.html#ClaimsParameter
	"claims",
	// https://openid.net/specs/openid-connect-core-1_0.html#JWTRequests
	"request",
	"request_uri",
	// https://openid.net/specs/openid-connect-core-1_0.html#RegistrationParameter
	"registration",
	// https://tools.ietf.org/html/rfc7636#section-4.3
	"code_challenge",
	"code_challenge_method",
}

func New(form url.Values, authnSession *auth.AuthnSession) *T {
	// We only keep parameters that is defined in the specs.
	sanitized := url.Values{}
	for _, paramName := range authenticationRequestParams {
		if _, ok := form[paramName]; ok {
			sanitized.Set(paramName, form.Get(paramName))
		}
	}
	return &T{
		Form:         sanitized,
		AuthnSession: authnSession,
	}
}
