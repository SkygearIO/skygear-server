package password

import (
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
	corePassword "github.com/skygeario/skygear-server/pkg/core/password"
	"github.com/skygeario/skygear-server/pkg/core/uuid"
)

type Principal struct {
	ID              string
	UserID          string
	LoginIDKey      string
	LoginID         string
	OriginalLoginID string
	UniqueKey       string
	Realm           string
	HashedPassword  []byte
	ClaimsValue     map[string]interface{}
}

func NewPrincipal() Principal {
	return Principal{
		ID: uuid.New(),
	}
}

func (p *Principal) setPassword(password string) (err error) {
	p.HashedPassword, err = corePassword.Hash([]byte(password))
	return
}

func (p *Principal) migratePassword(password string) (migrated bool, err error) {
	migrated, err = corePassword.TryMigrate([]byte(password), &p.HashedPassword)
	return
}

func (p *Principal) IsSamePassword(password string) bool {
	return corePassword.Compare([]byte(password), p.HashedPassword) == nil
}

func (p *Principal) VerifyPassword(password string) error {
	if !p.IsSamePassword(password) {
		return ErrInvalidCredentials
	}
	return nil
}

func (p *Principal) deriveClaims(checker loginid.Checker) {
	standardKey, hasStandardKey := checker.StandardKey(p.LoginIDKey)
	claimsValue := map[string]interface{}{}
	if hasStandardKey {
		claimsValue[string(standardKey)] = p.LoginID
	}
	p.ClaimsValue = claimsValue
}

func (p *Principal) PrincipalID() string {
	return p.ID
}

func (p *Principal) PrincipalUserID() string {
	return p.UserID
}

func (p *Principal) ProviderID() string {
	return string(coreAuth.PrincipalTypePassword)
}

func (p *Principal) Attributes() principal.Attributes {
	return principal.Attributes{
		"login_id_key": p.LoginIDKey,
		"login_id":     p.LoginID,
	}
}

func (p *Principal) Claims() principal.Claims {
	return p.ClaimsValue
}
