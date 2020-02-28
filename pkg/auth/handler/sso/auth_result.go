package sso

import (
	"net/http"
	"net/url"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authnsession"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/oauth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/sso"
	"github.com/skygeario/skygear-server/pkg/core/async"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

func AttachAuthResultHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/sso/auth_result", &AuthResultHandlerFactory{
		Dependency: authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type AuthResultHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f *AuthResultHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &AuthResultHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	return h.RequireAuthz(h, h)
}

type AuthResultHandler struct {
	TxContext            db.TxContext               `dependency:"TxContext"`
	RequireAuthz         handler.RequireAuthz       `dependency:"RequireAuthz"`
	OAuthAuthProvider    oauth.Provider             `dependency:"OAuthAuthProvider"`
	HookProvider         hook.Provider              `dependency:"HookProvider"`
	AuthnSessionProvider authnsession.Provider      `dependency:"AuthnSessionProvider"`
	Validator            *validation.Validator      `dependency:"Validator"`
	AuthInfoStore        authinfo.Store             `dependency:"AuthInfoStore"`
	UserProfileStore     userprofile.Store          `dependency:"UserProfileStore"`
	IdentityProvider     principal.IdentityProvider `dependency:"IdentityProvider"`
	TaskQueue            async.Queue                `dependency:"AsyncTaskQueue"`
	WelcomeEmailEnabled  bool                       `dependency:"WelcomeEmailEnabled"`
	URLPrefix            *url.URL                   `dependency:"URLPrefix"`
	SSOProvider          sso.Provider               `dependency:"SSOProvider"`
}

func (h *AuthResultHandler) ProvideAuthzPolicy() authz.Policy {
	return authz.PolicyFunc(policy.DenyNoAccessKey)
}

type AuthResultPayload struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeVerifier      string `json:"code_verifier"`
}

// @JSONSchema
const AuthResultRequestSchema = `
{
	"$id": "#AuthResultRequest",
	"type": "object",
	"properties": {
		"authorization_code": { "type": "string", "minLength": 1 },
		"code_verifier": { "type": "string", "minLength": 1 }
	},
	"required": ["authorization_code", "code_verifier"]
}
`

func (h *AuthResultHandler) DecodeRequest(w http.ResponseWriter, r *http.Request) (payload *AuthResultPayload, err error) {
	err = handler.BindJSONBody(r, w, h.Validator, "#AuthResultRequest", &payload)
	return
}

func (h *AuthResultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var result interface{}
	var err error

	payload, err := h.DecodeRequest(w, r)
	if err != nil {
		h.AuthnSessionProvider.WriteResponse(w, nil, err)
		return
	}

	err = hook.WithTx(h.HookProvider, h.TxContext, func() (err error) {
		result, err = h.Handle(payload)
		return
	})
	h.AuthnSessionProvider.WriteResponse(w, result, err)
}

func (h *AuthResultHandler) Handle(payload *AuthResultPayload) (result interface{}, err error) {
	code, err := h.SSOProvider.DecodeSkygearAuthorizationCode(payload.AuthorizationCode)
	if err != nil {
		return
	}

	err = h.SSOProvider.VerifyPKCE(code, payload.CodeVerifier)
	if err != nil {
		return
	}

	respHandler := respHandler{
		AuthnSessionProvider: h.AuthnSessionProvider,
		AuthInfoStore:        h.AuthInfoStore,
		OAuthAuthProvider:    h.OAuthAuthProvider,
		IdentityProvider:     h.IdentityProvider,
		UserProfileStore:     h.UserProfileStore,
		HookProvider:         h.HookProvider,
		WelcomeEmailEnabled:  h.WelcomeEmailEnabled,
		TaskQueue:            h.TaskQueue,
		URLPrefix:            h.URLPrefix,
	}

	return respHandler.CodeToResponse(code)
}
