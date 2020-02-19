package provider

import (
	"net/url"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/model"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

type ValidateProviderImpl struct {
	LoginIDKeys         []config.LoginIDKeyConfiguration
	Validator           *validation.Validator
	AuthContextProvider AuthContextProvider
}

func NewValidateProvider(
	tConfig *config.TenantConfiguration,
	validator *validation.Validator,
	authContextProvider AuthContextProvider,
) *ValidateProviderImpl {
	return &ValidateProviderImpl{
		LoginIDKeys:         tConfig.AppConfig.Auth.LoginIDKeys,
		Validator:           validator,
		AuthContextProvider: authContextProvider,
	}
}

func (p *ValidateProviderImpl) Prevalidate(form url.Values) {
	// Remove empty fields
	for name := range form {
		if form.Get(name) == "" {
			delete(form, name)
		}
	}

	// Set defaults
	if _, ok := form["x_login_id_input_type"]; !ok {
		if len(p.LoginIDKeys) > 0 {
			if string(p.LoginIDKeys[0].Type) == "phone" {
				form.Set("x_login_id_input_type", "phone")
			} else {
				form.Set("x_login_id_input_type", "text")
			}
		}
	}
}

func (p *ValidateProviderImpl) Validate(schemaID string, form url.Values) (map[string]interface{}, error) {
	j := make(map[string]interface{})
	// Do not support recurring parameter
	for name := range form {
		j[name] = form.Get(name)
	}
	err := p.Validator.ValidateGoValue(schemaID, j)

	// Validate client_id
	if err == nil {
		if _, ok := j["client_id"].(string); ok {
			accessKey := p.AuthContextProvider.AccessKey()
			if accessKey.Type != model.APIAccessKeyType {
				causes := []validation.ErrorCause{
					validation.ErrorCause{
						Kind:    validation.ErrorGeneral,
						Message: "invalid client_id",
						Pointer: "/client_id",
					},
				}
				err = validation.NewValidationFailed("validation failed", causes)
			}
		}
	}

	// TODO(authui): validate redirect_uri

	return j, err
}
