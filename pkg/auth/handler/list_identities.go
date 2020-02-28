package handler

import (
	"net/http"
	"sort"

	"github.com/skygeario/skygear-server/pkg/core/auth/principal"

	"github.com/skygeario/skygear-server/pkg/auth"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
)

func AttachListIdentitiesHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/identity/list", &ListIdentitiesHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type ListIdentitiesHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f ListIdentitiesHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &ListIdentitiesHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	return h.RequireAuthz(h, h)
}

// @JSONSchema
const IdentityListResponseSchema = `
{
	"$id": "#IdentityListResponse",
	"type": "object",
	"properties": {
		"result": {
			"type": "object",
			"properties": {
				"identities": { 
					"type": "array",
					"items": { "$ref": "#Identity" }
				}
			}
		}
	}
}
`

type IdentityListResponse struct {
	Identities []model.Identity `json:"identities"`
}

/*
	@Operation POST /identity/list - List identities
		Returns list of identities of current user.

		@Tag User
		@SecurityRequirement access_key
		@SecurityRequirement access_token

		@Response 200
			Current user and identity info.
			@JSONSchema {IdentityListResponse}
*/
type ListIdentitiesHandler struct {
	AuthContext      coreAuth.ContextGetter     `dependency:"AuthContextGetter"`
	RequireAuthz     handler.RequireAuthz       `dependency:"RequireAuthz"`
	TxContext        db.TxContext               `dependency:"TxContext"`
	IdentityProvider principal.IdentityProvider `dependency:"IdentityProvider"`
}

func (h ListIdentitiesHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		policy.RequireValidUser,
	)
}

func (h ListIdentitiesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result, err := h.Handle(w, r)
	if err == nil {
		handler.WriteResponse(w, handler.APIResponse{Result: result})
	} else {
		handler.WriteResponse(w, handler.APIResponse{Error: err})
	}
}

func (h ListIdentitiesHandler) Handle(w http.ResponseWriter, r *http.Request) (resp interface{}, err error) {
	if err = handler.DecodeJSONBody(r, w, &struct{}{}); err != nil {
		return
	}

	err = db.WithTx(h.TxContext, func() error {
		authInfo, _ := h.AuthContext.AuthInfo()

		principals, err := h.IdentityProvider.ListPrincipalsByUserID(authInfo.ID)
		if err != nil {
			return err
		}

		sort.Slice(principals, func(i, j int) bool {
			return principals[i].PrincipalID() < principals[j].PrincipalID()
		})

		identities := make([]model.Identity, len(principals))
		for i, p := range principals {
			identities[i] = model.NewIdentity(h.IdentityProvider, p)
		}

		resp = IdentityListResponse{Identities: identities}
		return nil
	})
	return
}
