package sso

import (
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authnsession"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/oauth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/sso"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/async"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

func AttachLoginHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/sso/{provider}/login", &LoginHandlerFactory{
		Dependency: authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type LoginHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f LoginHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &LoginHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	vars := mux.Vars(request)
	h.ProviderID = vars["provider"]
	h.OAuthProvider = h.ProviderFactory.NewOAuthProvider(h.ProviderID)
	return h.RequireAuthz(h, h)
}

// LoginRequestPayload login handler request payload
type LoginRequestPayload struct {
	AccessToken     string                `json:"access_token"`
	MergeRealm      string                `json:"-"`
	OnUserDuplicate model.OnUserDuplicate `json:"on_user_duplicate"`
}

func (p *LoginRequestPayload) SetDefaultValue() {
	if p.MergeRealm == "" {
		p.MergeRealm = loginid.DefaultRealm
	}
	if p.OnUserDuplicate == "" {
		p.OnUserDuplicate = model.OnUserDuplicateDefault
	}
}

// @JSONSchema
const LoginRequestSchema = `
{
	"$id": "#SSOLoginRequest",
	"type": "object",
	"properties": {
		"access_token": { "type": "string", "minLength": 1 },
		"on_user_duplicate": {"type": "string", "enum": ["abort", "merge", "create"] }
	},
	"required": ["access_token"]
}
`

/*
	@Operation POST /sso/{provider_id}/login - Login SSO provider with token
		Login the specified SSO provider, using access token obtained from the provider.

		@Tag SSO

		@Parameter {SSOProviderID}
		@RequestBody
			Describe the access token of SSO provider and login behavior.
			@JSONSchema {SSOLoginRequest}
		@Response 200 {EmptyResponse}

		@Callback user_create {UserSyncEvent}
		@Callback identity_create {UserSyncEvent}
		@Callback session_create {UserSyncEvent}
		@Callback user_sync {UserSyncEvent}
*/
type LoginHandler struct {
	TxContext            db.TxContext               `dependency:"TxContext"`
	Validator            *validation.Validator      `dependency:"Validator"`
	RequireAuthz         handler.RequireAuthz       `dependency:"RequireAuthz"`
	OAuthAuthProvider    oauth.Provider             `dependency:"OAuthAuthProvider"`
	IdentityProvider     principal.IdentityProvider `dependency:"IdentityProvider"`
	AuthInfoStore        authinfo.Store             `dependency:"AuthInfoStore"`
	AuthnSessionProvider authnsession.Provider      `dependency:"AuthnSessionProvider"`
	ProviderFactory      *sso.OAuthProviderFactory  `dependency:"SSOOAuthProviderFactory"`
	UserProfileStore     userprofile.Store          `dependency:"UserProfileStore"`
	HookProvider         hook.Provider              `dependency:"HookProvider"`
	WelcomeEmailEnabled  bool                       `dependency:"WelcomeEmailEnabled"`
	TaskQueue            async.Queue                `dependency:"AsyncTaskQueue"`
	URLPrefix            *url.URL                   `dependency:"URLPrefix"`
	SSOProvider          sso.Provider               `dependency:"SSOProvider"`
	OAuthProvider        sso.OAuthProvider
	ProviderID           string
}

func (h LoginHandler) ProvideAuthzPolicy() authz.Policy {
	return authz.PolicyFunc(policy.DenyNoAccessKey)
}

func (h LoginHandler) DecodeRequest(request *http.Request, resp http.ResponseWriter) (payload LoginRequestPayload, err error) {
	err = handler.BindJSONBody(request, resp, h.Validator, "#SSOLoginRequest", &payload)
	return
}

func (h LoginHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	var err error

	payload, err := h.DecodeRequest(req, resp)
	if err != nil {
		h.AuthnSessionProvider.WriteResponse(resp, nil, err)
		return
	}

	var result interface{}
	err = hook.WithTx(h.HookProvider, h.TxContext, func() (err error) {
		result, err = h.Handle(payload)
		return
	})
	h.AuthnSessionProvider.WriteResponse(resp, result, err)
}

func (h LoginHandler) Handle(payload LoginRequestPayload) (resp interface{}, err error) {
	if !h.SSOProvider.IsExternalAccessTokenFlowEnabled() {
		err = skyerr.NewNotFound("external access token flow is disabled")
		return
	}

	provider, ok := h.OAuthProvider.(sso.ExternalAccessTokenFlowProvider)
	if !ok {
		err = skyerr.NewNotFound("unknown provider")
		return
	}

	loginState := sso.LoginState{
		MergeRealm:      payload.MergeRealm,
		OnUserDuplicate: payload.OnUserDuplicate,
	}

	oauthAuthInfo, err := provider.ExternalAccessTokenGetAuthInfo(sso.NewBearerAccessTokenResp(payload.AccessToken))
	if err != nil {
		return
	}

	handler := respHandler{
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
	code, err := handler.LoginCode(oauthAuthInfo, "", loginState)
	if err != nil {
		return
	}

	resp, err = handler.CodeToResponse(code)
	if err != nil {
		return
	}

	return
}
