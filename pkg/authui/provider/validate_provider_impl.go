package provider

import (
	"crypto/subtle"
	"net/url"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

type ValidateProviderImpl struct {
	APIClients []config.APIClientConfiguration
	Validator  *validation.Validator
}

func NewValidateProvider(tConfig *config.TenantConfiguration, validator *validation.Validator) *ValidateProviderImpl {
	return &ValidateProviderImpl{
		APIClients: tConfig.AppConfig.Clients,
		Validator:  validator,
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
		if clientID, ok := j["client_id"].(string); ok {
			found := false
			for _, clientConfig := range p.APIClients {
				if subtle.ConstantTimeCompare([]byte(clientID), []byte(clientConfig.APIKey)) == 1 {
					found = true
				}
			}
			if !found {
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

	return j, err
}
