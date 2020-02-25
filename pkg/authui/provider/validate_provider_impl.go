package provider

import (
	"net/url"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/model"
	"github.com/skygeario/skygear-server/pkg/core/oauth"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

type ValidateProviderImpl struct {
	LoginIDKeys         []config.LoginIDKeyConfiguration
	OAuthConfiguration  *config.OAuthConfiguration
	Validator           *validation.Validator
	AuthContextProvider AuthContextProvider
}

var _ ValidateProvider = &ValidateProviderImpl{}

func NewValidateProvider(
	tConfig *config.TenantConfiguration,
	validator *validation.Validator,
	authContextProvider AuthContextProvider,
) *ValidateProviderImpl {
	return &ValidateProviderImpl{
		LoginIDKeys:         tConfig.AppConfig.Auth.LoginIDKeys,
		OAuthConfiguration:  tConfig.AppConfig.SSO.OAuth,
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

func (p *ValidateProviderImpl) Validate(schemaID string, form map[string]interface{}) (err error) {
	err = p.Validator.ValidateGoValue(schemaID, form)
	if err != nil {
		return
	}

	failWith := func(cause validation.ErrorCause) error {
		return validation.NewValidationFailed("validation failed", []validation.ErrorCause{
			cause,
		})
	}

	// Validate client_id
	if _, ok := form["client_id"].(string); ok {
		accessKey := p.AuthContextProvider.AccessKey()
		if accessKey.Type != model.APIAccessKeyType {
			err = failWith(validation.ErrorCause{
				Kind:    validation.ErrorGeneral,
				Message: "invalid client_id",
				Pointer: "/client_id",
			})
			return
		}
	}

	// Validate redirect_uri
	if redirectURI, ok := form["redirect_uri"].(string); ok {
		err = oauth.ValidateRedirectURI(p.OAuthConfiguration.AllowedCallbackURLs, redirectURI)
		if err != nil {
			err = failWith(validation.ErrorCause{
				Kind:    validation.ErrorGeneral,
				Message: "invalid redirect_uri",
				Pointer: "/redirect_uri",
			})
			return
		}
	}

	return nil
}
