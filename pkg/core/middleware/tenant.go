package middleware

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/model"
)

type ConfigurationProvider interface {
	ProvideConfig(r *http.Request) (config.TenantConfiguration, error)
}

type ConfigurationProviderFunc func(r *http.Request) (config.TenantConfiguration, error)

func (f ConfigurationProviderFunc) ProvideConfig(r *http.Request) (config.TenantConfiguration, error) {
	return f(r)
}

type TenantConfigurationMiddleware struct {
	ConfigurationProvider
}

func (m TenantConfigurationMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		configuration, err := m.ProvideConfig(r)
		if err != nil {
			http.Error(w, "Unable to retrieve configuration", http.StatusInternalServerError)
			return
		}

		// Tenant authentication
		// Set key type to header only, no rejection
		apiKey := model.GetAPIKey(r)
		apiKeyType := model.CheckAccessKeyType(configuration, apiKey)
		model.SetAccessKeyType(r, apiKeyType)

		// Tenant configuration
		config.SetTenantConfig(r, configuration)
		next.ServeHTTP(w, r)
	})
}
