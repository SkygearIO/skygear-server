package loginid

import (
	"strings"

	"golang.org/x/net/idna"
	"golang.org/x/text/secure/precis"
	"golang.org/x/text/unicode/norm"

	"github.com/skygeario/skygear-server/pkg/core/auth/metadata"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/errors"
)

type LoginIDNormalizer interface {
	Normalize(loginID string) (string, error)
	ComputeUniqueKey(normalizeLoginID string) (string, error)
}

type LoginIDNormalizerFactory interface {
	NewNormalizer(loginIDKey string) LoginIDNormalizer
}

func NewLoginIDNormalizerFactory(
	loginIDsKeys []config.LoginIDKeyConfiguration,
	loginIDTypes *config.LoginIDTypesConfiguration,
) *FactoryImpl {
	return &FactoryImpl{
		LoginIDsKeys: loginIDsKeys,
		LoginIDTypes: loginIDTypes,
	}
}

type FactoryImpl struct {
	LoginIDsKeys []config.LoginIDKeyConfiguration
	LoginIDTypes *config.LoginIDTypesConfiguration
}

func (f *FactoryImpl) NewNormalizer(loginIDKey string) LoginIDNormalizer {
	for _, c := range f.LoginIDsKeys {
		if c.Key == loginIDKey {
			return f.NewNormalizerWithType(c.Type)
		}
	}

	panic("password: invalid login id key: " + loginIDKey)
}

func (f *FactoryImpl) NewNormalizerWithType(loginIDKeyType config.LoginIDKeyType) LoginIDNormalizer {
	metadataKey, _ := loginIDKeyType.MetadataKey()
	switch metadataKey {
	case metadata.Email:
		return &LoginIDEmailNormalizer{
			config: f.LoginIDTypes.Email,
		}
	case metadata.Username:
		return &LoginIDUsernameNormalizer{
			config: f.LoginIDTypes.Username,
		}
	}

	return &LoginIDNullNormalizer{}
}

type LoginIDEmailNormalizer struct {
	config *config.LoginIDTypeEmailConfiguration
}

func (n *LoginIDEmailNormalizer) Normalize(loginID string) (string, error) {
	// refs from stdlib
	// https://golang.org/src/net/mail/message.go?s=5217:5250#L172
	at := strings.LastIndex(loginID, "@")
	if at < 0 {
		panic("password: malformed address, should be rejected by the email format checker")
	}
	local, domain := loginID[:at], loginID[at+1:]

	// convert the domain part
	var err error
	p := precis.NewFreeform(precis.FoldCase())
	domain, err = p.String(domain)
	if err != nil {
		return "", errors.HandledWithMessage(err, "failed to case fold email")
	}

	// convert the local part
	local = norm.NFKC.String(local)

	if !*n.config.CaseSensitive {
		local, err = p.String(local)
		if err != nil {
			return "", errors.HandledWithMessage(err, "failed to case fold email")
		}
	}

	if *n.config.IgnoreDotSign {
		local = strings.Replace(local, ".", "", -1)
	}

	return local + "@" + domain, nil
}

func (n *LoginIDEmailNormalizer) ComputeUniqueKey(normalizeLoginID string) (string, error) {
	at := strings.LastIndex(normalizeLoginID, "@")
	if at < 0 {
		panic("password: malformed address, should be rejected by the email format checker")
	}
	local, domain := normalizeLoginID[:at], normalizeLoginID[at+1:]
	punycode, err := idna.ToASCII(domain)
	if err != nil {
		return "", err
	}
	domain = punycode
	return local + "@" + domain, nil
}

type LoginIDUsernameNormalizer struct {
	config *config.LoginIDTypeUsernameConfiguration
}

func (n *LoginIDUsernameNormalizer) Normalize(loginID string) (string, error) {
	loginID = norm.NFKC.String(loginID)

	var err error
	if !*n.config.CaseSensitive {
		p := precis.NewIdentifier(precis.FoldCase())
		loginID, err = p.String(loginID)
		if err != nil {
			return "", errors.HandledWithMessage(err, "failed to case fold username")
		}
	}

	return loginID, nil
}

func (n *LoginIDUsernameNormalizer) ComputeUniqueKey(normalizeLoginID string) (string, error) {
	return normalizeLoginID, nil
}

type LoginIDNullNormalizer struct{}

func (n *LoginIDNullNormalizer) Normalize(loginID string) (string, error) {
	return loginID, nil
}

func (n *LoginIDNullNormalizer) ComputeUniqueKey(normalizeLoginID string) (string, error) {
	return normalizeLoginID, nil
}

// this ensures that our structure conform to certain interfaces.
var (
	_ LoginIDNormalizer = &LoginIDEmailNormalizer{}
	_ LoginIDNormalizer = &LoginIDUsernameNormalizer{}
	_ LoginIDNormalizer = &LoginIDNullNormalizer{}
)
