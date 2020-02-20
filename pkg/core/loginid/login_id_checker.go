package loginid

import (
	"fmt"

	"github.com/skygeario/skygear-server/pkg/core/auth/metadata"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

type Checker interface {
	ValidateOne(loginID LoginID) error
	Validate(loginIDs []LoginID) error
	CheckType(loginIDKey string, standardKey metadata.StandardKey) bool
	StandardKey(loginIDKey string) (metadata.StandardKey, bool)
}

func NewDefaultChecker(
	loginIDsKeys []config.LoginIDKeyConfiguration,
	loginIDTypes *config.LoginIDTypesConfiguration,
	reservedNameChecker *ReservedNameChecker,
) *DefaultChecker {
	return &DefaultChecker{
		LoginIDsKeys: loginIDsKeys,
		LoginIDTypes: loginIDTypes,
		LoginIDTypeCheckerFactory: NewTypeCheckerFactory(
			loginIDsKeys,
			loginIDTypes,
			reservedNameChecker,
		),
	}
}

type DefaultChecker struct {
	LoginIDsKeys              []config.LoginIDKeyConfiguration
	LoginIDTypes              *config.LoginIDTypesConfiguration
	LoginIDTypeCheckerFactory TypeCheckerFactory
}

var _ Checker = &DefaultChecker{}

func (c *DefaultChecker) Validate(loginIDs []LoginID) error {
	amounts := map[string]int{}
	for i, loginID := range loginIDs {
		amounts[loginID.Key]++

		if err := c.ValidateOne(loginID); err != nil {
			if causes := validation.ErrorCauses(err); len(causes) > 0 {
				for j := range causes {
					causes[j].Pointer = fmt.Sprintf("/%d%s", i, causes[j].Pointer)
				}
				err = validation.NewValidationFailed("invalid login IDs", causes)
			}
			return err
		}
	}

	for _, keyConfig := range c.LoginIDsKeys {
		amount := amounts[keyConfig.Key]
		if amount > *keyConfig.Maximum {
			return validation.NewValidationFailed("invalid login IDs", []validation.ErrorCause{{
				Kind:    validation.ErrorEntryAmount,
				Pointer: "",
				Message: "too many login IDs",
				Details: map[string]interface{}{"key": keyConfig.Key, "lte": *keyConfig.Maximum},
			}})
		}
	}

	if len(loginIDs) == 0 {
		return validation.NewValidationFailed("invalid login IDs", []validation.ErrorCause{{
			Kind:    validation.ErrorRequired,
			Pointer: "",
			Message: "login ID is required",
		}})
	}

	return nil
}

func (c *DefaultChecker) ValidateOne(loginID LoginID) error {
	allowed := false
	var loginIDType config.LoginIDKeyType
	for _, keyConfig := range c.LoginIDsKeys {
		if keyConfig.Key == loginID.Key {
			loginIDType = keyConfig.Type
			allowed = true
		}
	}
	if !allowed {
		return validation.NewValidationFailed("invalid login ID", []validation.ErrorCause{{
			Kind:    validation.ErrorGeneral,
			Pointer: "/key",
			Message: "login ID key is not allowed",
		}})
	}

	if loginID.Value == "" {
		return validation.NewValidationFailed("invalid login ID", []validation.ErrorCause{{
			Kind:    validation.ErrorRequired,
			Pointer: "/value",
			Message: "login ID is required",
		}})
	}

	if err := c.LoginIDTypeCheckerFactory.NewChecker(loginIDType).Validate(loginID.Value); err != nil {
		return err
	}

	return nil
}

func (c *DefaultChecker) StandardKey(loginIDKey string) (key metadata.StandardKey, ok bool) {
	var config config.LoginIDKeyConfiguration
	for _, keyConfig := range c.LoginIDsKeys {
		if keyConfig.Key == loginIDKey {
			config = keyConfig
			ok = true
			break
		}
	}
	if !ok {
		return
	}

	key, ok = config.Type.MetadataKey()
	return
}

func (c *DefaultChecker) CheckType(loginIDKey string, standardKey metadata.StandardKey) bool {
	loginIDKeyStandardKey, ok := c.StandardKey(loginIDKey)
	return ok && loginIDKeyStandardKey == standardKey
}
