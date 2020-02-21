package password

import (
	"time"

	"github.com/sirupsen/logrus"

	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/metadata"
	"github.com/skygeario/skygear-server/pkg/core/auth/passwordhistory"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
)

var (
	timeNow = func() time.Time { return time.Now().UTC() }
)

type ProviderImpl struct {
	store                    Store
	logger                   *logrus.Entry
	loginIDsKeys             []config.LoginIDKeyConfiguration
	loginIDChecker           loginid.Checker
	loginIDNormalizerFactory loginid.NormalizerFactory
	realmChecker             loginid.RealmChecker
	allowedRealms            []string
	passwordHistoryEnabled   bool
	passwordHistoryStore     passwordhistory.Store
}

func newProvider(
	passwordStore Store,
	passwordHistoryStore passwordhistory.Store,
	loggerFactory logging.Factory,
	loginIDsKeys []config.LoginIDKeyConfiguration,
	loginIDTypes *config.LoginIDTypesConfiguration,
	allowedRealms []string,
	passwordHistoryEnabled bool,
	reservedNameChecker *loginid.ReservedNameChecker,
) *ProviderImpl {
	return &ProviderImpl{
		store:        passwordStore,
		logger:       loggerFactory.NewLogger("password-provider"),
		loginIDsKeys: loginIDsKeys,
		loginIDChecker: loginid.NewDefaultChecker(
			loginIDsKeys,
			loginIDTypes,
			reservedNameChecker,
		),
		realmChecker: &loginid.DefaultRealmChecker{
			AllowedRealms: allowedRealms,
		},
		loginIDNormalizerFactory: loginid.NewNormalizerFactory(loginIDsKeys, loginIDTypes),
		allowedRealms:            allowedRealms,
		passwordHistoryEnabled:   passwordHistoryEnabled,
		passwordHistoryStore:     passwordHistoryStore,
	}
}

func NewProvider(
	passwordStore Store,
	passwordHistoryStore passwordhistory.Store,
	loggerFactory logging.Factory,
	loginIDsKeys []config.LoginIDKeyConfiguration,
	loginIDTypes *config.LoginIDTypesConfiguration,
	allowedRealms []string,
	passwordHistoryEnabled bool,
	reservedNameChecker *loginid.ReservedNameChecker,
) *ProviderImpl {
	return newProvider(passwordStore, passwordHistoryStore, loggerFactory, loginIDsKeys, loginIDTypes, allowedRealms, passwordHistoryEnabled, reservedNameChecker)
}

func (p *ProviderImpl) ValidateLoginID(loginID loginid.LoginID) error {
	return p.loginIDChecker.ValidateOne(loginID)
}

func (p *ProviderImpl) ValidateLoginIDs(loginIDs []loginid.LoginID) error {
	return p.loginIDChecker.Validate(loginIDs)
}

func (p *ProviderImpl) CheckLoginIDKeyType(loginIDKey string, standardKey metadata.StandardKey) bool {
	return p.loginIDChecker.CheckType(loginIDKey, standardKey)
}

func (p *ProviderImpl) IsRealmValid(realm string) bool {
	return p.realmChecker.IsValid(realm)
}

func (p *ProviderImpl) IsDefaultAllowedRealms() bool {
	return len(p.allowedRealms) == 1 && p.allowedRealms[0] == loginid.DefaultRealm
}

func (p *ProviderImpl) MakePrincipal(userID string, password string, loginID loginid.LoginID, realm string) (*Principal, error) {
	normalizer := p.loginIDNormalizerFactory.NewNormalizer(loginID.Key)
	loginIDValue := loginID.Value
	normalizedloginIDValue, err := normalizer.Normalize(loginID.Value)
	if err != nil {
		return nil, errors.HandledWithMessage(err, "failed to normalized login id")
	}

	uniqueKey, err := normalizer.ComputeUniqueKey(normalizedloginIDValue)
	if err != nil {
		return nil, errors.HandledWithMessage(err, "failed to compute login id unique key")
	}

	principal := NewPrincipal()
	principal.UserID = userID
	principal.LoginIDKey = loginID.Key
	principal.LoginID = normalizedloginIDValue
	principal.OriginalLoginID = loginIDValue
	principal.UniqueKey = uniqueKey
	principal.Realm = realm
	principal.deriveClaims(p.loginIDChecker)
	err = principal.setPassword(password)
	if err != nil {
		return nil, errors.HandledWithMessage(err, "failed to set password")
	}

	return &principal, nil
}

func (p *ProviderImpl) CreatePrincipalsByLoginID(userID string, password string, loginIDs []loginid.LoginID, realm string) ([]*Principal, error) {
	var principals []*Principal
	for _, loginID := range loginIDs {
		principal, err := p.MakePrincipal(userID, password, loginID, realm)
		if err != nil {
			return nil, err
		}

		err = p.CreatePrincipal(principal)
		if err != nil {
			if !errors.Is(err, ErrLoginIDAlreadyUsed) {
				err = errors.HandledWithMessage(err, "failed to create principal")
			}
			return nil, err
		}

		principals = append(principals, principal)
	}

	return principals, nil
}

func (p *ProviderImpl) CreatePrincipal(principal *Principal) (err error) {
	// Create principal
	err = p.store.CreatePrincipal(principal)
	if err != nil {
		return
	}

	err = p.savePasswordHistory(principal)

	return
}

func (p *ProviderImpl) DeletePrincipal(principal *Principal) error {
	err := p.store.DeletePrincipal(principal)
	if err != nil {
		return err
	}
	return nil
}

func (p *ProviderImpl) savePasswordHistory(principal *Principal) error {
	if p.passwordHistoryEnabled {
		err := p.passwordHistoryStore.CreatePasswordHistory(
			principal.UserID, principal.HashedPassword, timeNow(),
		)
		if err != nil {
			return errors.Newf("failed to create password history: %w", err)
		}
	}
	return nil
}

func (p *ProviderImpl) GetPrincipalByLoginIDWithRealm(loginIDKey string, loginID string, realm string, pp *Principal) (err error) {
	var principals []*Principal
	for _, loginIDKeyConfig := range p.loginIDsKeys {
		if loginIDKey == "" || loginIDKeyConfig.Key == loginIDKey {
			invalid := p.loginIDChecker.ValidateOne(loginid.LoginID{
				Key:   loginIDKeyConfig.Key,
				Value: loginID,
			})
			if invalid != nil {
				continue
			}

			normalizer := p.loginIDNormalizerFactory.NewNormalizer(loginIDKeyConfig.Key)
			normalizedloginID, e := normalizer.Normalize(loginID)
			if e != nil {
				err = errors.HandledWithMessage(e, "failed to normalized login id")
				return
			}
			ps, e := p.store.GetPrincipals(loginIDKeyConfig.Key, normalizedloginID, &realm)
			if e != nil {
				err = errors.HandledWithMessage(e, "failed to get principal by login ID & realm")
				return
			}
			if len(ps) > 0 {
				principals = append(principals, ps...)
			}
		}
	}

	if len(principals) == 0 {
		err = principal.ErrNotFound
	} else if len(principals) > 1 {
		err = principal.ErrMultipleResultsFound
	} else {
		*pp = *principals[0]
	}

	return
}

func (p *ProviderImpl) GetPrincipalsByUserID(userID string) (principals []*Principal, err error) {
	return p.store.GetPrincipalsByUserID(userID)
}

func (p *ProviderImpl) GetPrincipalsByLoginID(loginIDKey string, loginID string) (principals []*Principal, err error) {
	var result []*Principal
	for _, loginIDKeyConfig := range p.loginIDsKeys {
		if loginIDKey == "" || loginIDKeyConfig.Key == loginIDKey {
			normalizer := p.loginIDNormalizerFactory.NewNormalizer(loginIDKeyConfig.Key)
			normalizedloginID, e := normalizer.Normalize(loginID)
			if e != nil {
				err = errors.HandledWithMessage(e, "failed to normalized login id")
				return
			}
			ps, e := p.store.GetPrincipals(loginIDKeyConfig.Key, normalizedloginID, nil)
			if e != nil {
				err = errors.HandledWithMessage(e, "failed to get principal by login ID")
				return
			}
			if len(ps) > 0 {
				result = append(result, ps...)
			}
		}
	}

	principals = result
	return
}

func (p *ProviderImpl) UpdatePassword(principal *Principal, password string) (err error) {
	var isPasswordChanged = !principal.IsSamePassword(password)

	err = principal.setPassword(password)
	if err != nil {
		err = errors.HandledWithMessage(err, "failed to update password")
		return
	}

	err = p.store.UpdatePassword(principal, password)
	if err != nil {
		err = errors.HandledWithMessage(err, "failed to update password")
		return
	}

	if isPasswordChanged {
		err = p.savePasswordHistory(principal)
		if err != nil {
			err = errors.HandledWithMessage(err, "failed to update password")
			return
		}
	}

	return
}

func (p *ProviderImpl) MigratePassword(principal *Principal, password string) (err error) {
	migrated, err := principal.migratePassword(password)
	if err != nil {
		err = errors.HandledWithMessage(err, "failed to migrate password")
		return err
	}
	if !migrated {
		return
	}

	err = p.store.UpdatePassword(principal, password)
	if err != nil {
		err = errors.HandledWithMessage(err, "failed to update password")
	}

	return
}

func (p *ProviderImpl) ID() string {
	return string(coreAuth.PrincipalTypePassword)
}

func (p *ProviderImpl) GetPrincipalByID(principalID string) (principal.Principal, error) {
	return p.store.GetPrincipalByID(principalID)
}

func (p *ProviderImpl) ListPrincipalsByClaim(claimName string, claimValue string) ([]principal.Principal, error) {
	principals, err := p.store.GetPrincipalsByClaim(claimName, claimValue)
	if err != nil {
		return nil, err
	}

	genericPrincipals := []principal.Principal{}
	for _, principal := range principals {
		genericPrincipals = append(genericPrincipals, principal)
	}

	return genericPrincipals, nil
}

func (p *ProviderImpl) ListPrincipalsByUserID(userID string) ([]principal.Principal, error) {
	principals, err := p.store.GetPrincipalsByUserID(userID)
	if err != nil {
		return nil, err
	}

	genericPrincipals := []principal.Principal{}
	for _, principal := range principals {
		genericPrincipals = append(genericPrincipals, principal)
	}

	return genericPrincipals, nil
}

// this ensures that our structure conform to certain interfaces.
var (
	_ Provider = &ProviderImpl{}
)
