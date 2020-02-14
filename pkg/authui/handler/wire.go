//+build wireinject

package handler

import (
	"net/http"

	"github.com/google/wire"

	"github.com/skygeario/skygear-server/pkg/core/config"
	coreTemplate "github.com/skygeario/skygear-server/pkg/core/template"

	"github.com/skygeario/skygear-server/pkg/authui/inject"
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

var DefaultSet = wire.NewSet(
	ProvideTenantConfig,
	ProvideAssetGearLoader,
	ProvideEnableFileSystemTemplate,
)

func InjectRootHandler(r *http.Request) *RootHandler {
	wire.Build(NewRootHandler)
	return &RootHandler{}
}

func InjectAuthorizeHandler(r *http.Request, dep *inject.BootTimeDependency) *AuthorizeHandler {
	wire.Build(
		DefaultSet,
		template.NewEngine,
		NewAuthorizeHandler,
	)
	return &AuthorizeHandler{}
}
