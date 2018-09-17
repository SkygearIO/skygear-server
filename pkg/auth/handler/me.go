package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
)

func AttachMeHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/me", &MeHandlerFactory{
		authDependency,
	}).Methods("POST")
	return server
}

type MeHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f MeHandlerFactory) NewHandler(ctx context.Context, tenantConfig config.TenantConfiguration) handler.Handler {
	h := &MeHandler{}
	inject.DefaultInject(h, f.Dependency, ctx, tenantConfig)
	return h
}

// MeHandler handles me request
type MeHandler struct{}

func (h MeHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		authz.PolicyFunc(policy.RequireAuthenticated),
		authz.PolicyFunc(policy.DenyDisabledUser),
	)
}

func (h MeHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request, ctx handler.AuthContext) {
	output, err := json.Marshal(ctx.AuthInfo)
	if err != nil {
		// TODO:
		// handle error properly
		panic(err)
	}

	fmt.Fprint(rw, string(output))
}
