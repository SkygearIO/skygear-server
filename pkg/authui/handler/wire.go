//+build wireinject

package handler

import (
	"net/http"

	"github.com/google/wire"

	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/config"
	coreTemplate "github.com/skygeario/skygear-server/pkg/core/template"
	"github.com/skygeario/skygear-server/pkg/core/validation"

	"github.com/skygeario/skygear-server/pkg/authui/inject"
	"github.com/skygeario/skygear-server/pkg/authui/provider"
	"github.com/skygeario/skygear-server/pkg/authui/template"
)

func ProvideTenantConfig(r *http.Request) *config.TenantConfiguration {
	return config.GetTenantConfig(r.Context())
}

func ProvideAssetGearLoader(dep *inject.BootTimeDependency) *coreTemplate.AssetGearLoader {
	configuration := dep.Configuration
	if configuration.Template.AssetGearEndpoint != "" && configuration.Template.AssetGearMasterKey != "" {
		return &coreTemplate.AssetGearLoader{
			AssetGearEndpoint:  configuration.Template.AssetGearEndpoint,
			AssetGearMasterKey: configuration.Template.AssetGearMasterKey,
		}
	}
	return nil
}

func ProvideEnableFileSystemTemplate(dep *inject.BootTimeDependency) inject.EnableFileSystemTemplate {
	return inject.EnableFileSystemTemplate(dep.Configuration.Template.EnableFileLoader)
}

func ProvideValidator(dep *inject.BootTimeDependency) *validation.Validator {
	return dep.Validator
}

var DefaultSet = wire.NewSet(
	ProvideTenantConfig,
	ProvideAssetGearLoader,
	ProvideEnableFileSystemTemplate,
	ProvideValidator,

	template.NewEngine,

	wire.Bind(new(provider.RenderProvider), new(*provider.RenderProviderImpl)),
	provider.NewRenderProvider,

	wire.Bind(new(provider.ValidateProvider), new(*provider.ValidateProviderImpl)),
	provider.NewValidateProvider,

	wire.Bind(new(coreAuth.ContextGetter), new(*provider.AuthContextProviderImpl)),
	wire.Bind(new(provider.AuthContextProvider), new(*provider.AuthContextProviderImpl)),
	provider.NewAuthContextProvider,
)

func InjectRootHandler(r *http.Request) *RootHandler {
	wire.Build(NewRootHandler)
	return &RootHandler{}
}

func InjectAuthorizeHandler(r *http.Request, dep *inject.BootTimeDependency) *AuthorizeHandler {
	wire.Build(DefaultSet, NewAuthorizeHandler)
	return &AuthorizeHandler{}
}
