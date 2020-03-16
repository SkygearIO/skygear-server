// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package auth

import (
	"github.com/gorilla/mux"
	"github.com/skygeario/skygear-server/pkg/core/auth"
	"net/http"
)

// Injectors from wire.go:

func NewAccessKeyMiddleware(r *http.Request, m DependencyMap) mux.MiddlewareFunc {
	context := ProvideContext(r)
	tenantConfiguration := ProvideTenantConfig(context)
	middlewareFunc := auth.ProvideAccessKeyMiddleware(tenantConfiguration)
	return middlewareFunc
}