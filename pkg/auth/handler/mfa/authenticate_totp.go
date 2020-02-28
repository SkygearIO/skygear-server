package mfa

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authnsession"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/mfa"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

func AttachAuthenticateTOTPHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/mfa/totp/authenticate", &AuthenticateTOTPHandlerFactory{
		Dependency: authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type AuthenticateTOTPHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f AuthenticateTOTPHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &AuthenticateTOTPHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	return h.RequireAuthz(h, h)
}

type AuthenticateTOTPRequest struct {
	AuthnSessionToken  string `json:"authn_session_token"`
	OTP                string `json:"otp"`
	RequestBearerToken bool   `json:"request_bearer_token"`
}

// @JSONSchema
const AuthenticateTOTPRequestSchema = `
{
	"$id": "#AuthenticateTOTPRequest",
	"type": "object",
	"properties": {
		"authn_session_token": { "type": "string", "minLength": 1 },
		"otp": { "type": "string", "minLength": 1 },
		"request_bearer_token": { "type": "boolean" }
	},
	"required": ["otp"]
}
`

/*
	@Operation POST /mfa/totp/authenticate - Authenticate with TOTP authenticator.
		Authenticate with TOTP authenticator.

		@Tag User
		@SecurityRequirement access_key

		@RequestBody
			@JSONSchema {AuthenticateTOTPRequest}
		@Response 200
			Logged in user and access token.
			@JSONSchema {AuthResponse}

		@Callback session_create {SessionCreateEvent}
		@Callback user_sync {UserSyncEvent}
*/
type AuthenticateTOTPHandler struct {
	TxContext            db.TxContext            `dependency:"TxContext"`
	Validator            *validation.Validator   `dependency:"Validator"`
	AuthContext          coreAuth.ContextGetter  `dependency:"AuthContextGetter"`
	RequireAuthz         handler.RequireAuthz    `dependency:"RequireAuthz"`
	SessionProvider      session.Provider        `dependency:"SessionProvider"`
	MFAProvider          mfa.Provider            `dependency:"MFAProvider"`
	MFAConfiguration     config.MFAConfiguration `dependency:"MFAConfiguration"`
	HookProvider         hook.Provider           `dependency:"HookProvider"`
	AuthnSessionProvider authnsession.Provider   `dependency:"AuthnSessionProvider"`
}

func (h *AuthenticateTOTPHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		authz.PolicyFunc(policy.DenyInvalidSession),
	)
}

func (h *AuthenticateTOTPHandler) DecodeRequest(request *http.Request, resp http.ResponseWriter) (AuthenticateTOTPRequest, error) {
	payload := AuthenticateTOTPRequest{}
	err := handler.BindJSONBody(request, resp, h.Validator, "#AuthenticateTOTPRequest", &payload)
	return payload, err
}

func (h *AuthenticateTOTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	payload, err := h.DecodeRequest(r, w)
	if err != nil {
		h.AuthnSessionProvider.WriteResponse(w, nil, err)
		return
	}

	result, err := handler.Transactional(h.TxContext, func() (result interface{}, err error) {
		result, err = h.Handle(payload)
		if err == nil {
			err = h.HookProvider.WillCommitTx()
		}
		return
	})
	if err == nil {
		h.HookProvider.DidCommitTx()
	}
	h.AuthnSessionProvider.WriteResponse(w, result, err)
}

func (h *AuthenticateTOTPHandler) Handle(req interface{}) (resp interface{}, err error) {
	payload := req.(AuthenticateTOTPRequest)

	userID, sess, authnSess, err := h.AuthnSessionProvider.Resolve(h.AuthContext, payload.AuthnSessionToken, authnsession.ResolveOptions{
		MFAOption: authnsession.ResolveMFAOptionAlwaysAccept,
	})
	if err != nil {
		return
	}

	a, bearerToken, err := h.MFAProvider.AuthenticateTOTP(userID, payload.OTP, payload.RequestBearerToken)
	if err != nil {
		return
	}
	opts := coreAuth.AuthnSessionStepMFAOptions{
		AuthenticatorID:          a.ID,
		AuthenticatorType:        a.Type,
		AuthenticatorBearerToken: bearerToken,
	}

	if sess != nil {
		err = h.SessionProvider.UpdateMFA(sess, opts)
		if err != nil {
			return
		}
		resp, err = h.AuthnSessionProvider.GenerateResponseWithSession(sess, bearerToken)
		if err != nil {
			return
		}
	} else if authnSess != nil {
		err = h.MFAProvider.StepMFA(authnSess, opts)
		if err != nil {
			return
		}
		resp, err = h.AuthnSessionProvider.GenerateResponseAndUpdateLastLoginAt(*authnSess)
		if err != nil {
			return
		}
	}

	return
}
