//+build wireinject

package handler

import (
	"net/http"

	"github.com/google/wire"

	"github.com/skygeario/skygear-server/pkg/core/config"
)

func ProvideTenantConfig(r *http.Request) *config.TenantConfiguration {
	return config.GetTenantConfig(r.Context())
}

func InjectRootHandler(r *http.Request) *RootHandler {
	wire.Build(NewRootHandler)
	return &RootHandler{}
}
