package oauth

import (
	"net/url"
	"strings"

	"github.com/skygeario/skygear-server/pkg/core/errors"
)

// ValidateRedirectURI assumes allowedRedirectURIs are URL without query or fragment.
// It removes query or fragment of redirectURI and check if it appears in allowedRedirectURIs.
// It also ignore trailing slash in allowedRedirectURIs and redirectURI.
func ValidateRedirectURI(allowedRedirectURIs []string, redirectURI string) (err error) {
	// The logic of this function must be in sync with the inline javascript implementation.
	if redirectURI == "" {
		err = errors.New("missing redirect URI")
		return
	}

	u, err := url.Parse(redirectURI)
	if err != nil {
		err = errors.New("invalid redirect URI")
		return
	}

	u.RawQuery = ""
	u.Fragment = ""
	redirectURI = u.String()

	redirectURI = strings.TrimSuffix(redirectURI, "/")
	for _, v := range allowedRedirectURIs {
		allowed := strings.TrimSuffix(v, "/")
		if redirectURI == allowed {
			return nil
		}
	}

	err = errors.New("redirect URI is not whitelisted")
	return
}
