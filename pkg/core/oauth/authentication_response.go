package oauth

import (
	"net/url"
)

// NewAuthenticationResponse implements
// https://openid.net/specs/openid-connect-core-1_0.html#AuthResponse
func NewAuthenticationResponse(form url.Values, code string) (u *url.URL, err error) {
	u, err = url.Parse(form.Get("redirect_uri"))
	if err != nil {
		return
	}

	query := u.Query()
	if state := form.Get("state"); state != "" {
		query.Set("state", state)
	}
	query.Set("code", code)
	u.RawQuery = query.Encode()

	return
}
