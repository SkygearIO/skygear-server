// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package handler

import (
	"github.com/google/wire"
	"github.com/skygeario/skygear-server/pkg/authui/inject"
	"github.com/skygeario/skygear-server/pkg/authui/template"
	"github.com/skygeario/skygear-server/pkg/core/config"
	template2 "github.com/skygeario/skygear-server/pkg/core/template"
	"net/http"
)

// Injectors from wire.go:

func InjectRootHandler(r *http.Request) *RootHandler {
	rootHandler := NewRootHandler()
	return rootHandler
}

func InjectAuthorizeHandler(r *http.Request, dep *inject.BootTimeDependency) *AuthorizeHandler {
	tenantConfiguration := ProvideTenantConfig(r)
	enableFileSystemTemplate := ProvideEnableFileSystemTemplate(dep)
	assetGearLoader := ProvideAssetGearLoader(dep)
	engine := template.NewEngine(tenantConfiguration, enableFileSystemTemplate, assetGearLoader)
	authorizeHandler := NewAuthorizeHandler(engine)
	return authorizeHandler
}

// wire.go:

func ProvideTenantConfig(r *http.Request) *config.TenantConfiguration {
	return config.GetTenantConfig(r.Context())
}

func ProvideAssetGearLoader(dep *inject.BootTimeDependency) *template2.AssetGearLoader {
	configuration := dep.Configuration
	if configuration.Template.AssetGearEndpoint != "" && configuration.Template.AssetGearMasterKey != "" {
		return &template2.AssetGearLoader{
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
