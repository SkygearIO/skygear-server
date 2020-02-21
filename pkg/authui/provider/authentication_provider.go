package provider

import (
	"fmt"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"net/url"
	"regexp"
)

type AuthenticationProvider interface {
	// AuthenticateWithPassword creates a new AuthnSession.
	AuthenticateWithPassword(loginID string, password string) (*coreAuth.AuthnSession, error)

	// FromToken restores a AuthnSession from a token.
	FromToken(token string) (*coreAuth.AuthnSession, error)

	// ToToken stores a AuthnSession as a token.
	ToToken(*coreAuth.AuthnSession) (string, error)
}

var reKeepDigit = regexp.MustCompile(`[^0-9]`)

// DeriveLoginID derives login ID from
// x_login_id_input_type, x_login_id, x_calling_code, x_national_number.
func DeriveLoginID(form url.Values) string {
	switch form.Get("x_login_id_input_type") {
	case "phone":
		return fmt.Sprintf("+%s%s", form.Get("x_calling_code"), reKeepDigit.ReplaceAllString(form.Get("x_national_number"), ""))
	default:
		return form.Get("x_login_id")
	}
}
