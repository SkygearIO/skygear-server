package handler

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/skygeario/skygear-server/pkg/core/config"

	"github.com/skygeario/skygear-server/pkg/authui/inject"
	"github.com/skygeario/skygear-server/pkg/authui/provider"
	"github.com/skygeario/skygear-server/pkg/authui/template"
)

type AuthorizeHandler struct {
	ValidateProvider       provider.ValidateProvider
	RenderProvider         provider.RenderProvider
	AuthContextProvider    provider.AuthContextProvider
	AuthenticationProvider provider.AuthenticationProvider
}

func NewAuthorizeHandler(
	validateProvider provider.ValidateProvider,
	renderProvider provider.RenderProvider,
	authContextProvider provider.AuthContextProvider,
	authenticationProvider provider.AuthenticationProvider,
) *AuthorizeHandler {
	return &AuthorizeHandler{
		ValidateProvider:       validateProvider,
		RenderProvider:         renderProvider,
		AuthContextProvider:    authContextProvider,
		AuthenticationProvider: authenticationProvider,
	}
}

func AttachAuthorizeHandler(router *mux.Router, dep *inject.BootTimeDependency) {
	router.Path("/authorize").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		InjectAuthorizeHandler(r, dep).ServeHTTP(w, r)
	}).
		// https://openid.net/specs/openid-connect-core-1_0.html#AuthRequest
		// Must support either POST or GET and the actual HTTP method
		// does not matter
		Methods("POST", "GET")
}

const AuthorizeRequestSchema = `
{
	"$id": "#AuthorizeRequest",
	"type": "object",
	"properties": {
		"scope": { "type": "string", "const": "openid" },
		"response_type": { "type": "string", "const": "code" },
		"client_id": { "type": "string" },
		"redirect_uri": { "type": "string" },
		"code_challenge_method": { "type": "string", "const": "S256" },
		"code_challenge": { "type": "string" },
		"x_login_id_input_type": { "type": "string", "enum": ["phone", "text"] }
	},
	"required": ["scope", "response_type", "client_id", "redirect_uri", "code_challenge_method", "code_challenge"]
}
`

const AuthorizeLoginIDRequestSchema = `
{
	"$id": "#AuthorizeLoginIDRequest",
	"type": "object",
	"properties": {
		"scope": { "type": "string", "const": "openid" },
		"response_type": { "type": "string", "const": "code" },
		"client_id": { "type": "string" },
		"redirect_uri": { "type": "string" },
		"code_challenge_method": { "type": "string", "const": "S256" },
		"code_challenge": { "type": "string" },
		"x_login_id_input_type": { "type": "string", "enum": ["phone", "text"] },
		"x_step": { "type": "string", "const": "submit_login_id" },
		"x_calling_code": { "type": "string" },
		"x_national_number": { "type": "string" },
		"x_login_id": { "type": "string" }
	},
	"required": ["scope", "response_type", "client_id", "redirect_uri", "code_challenge_method", "code_challenge", "x_login_id_input_type", "x_step"],
	"oneOf": [
	{
		"properties": {
			"x_login_id_input_type": { "type": "string", "const": "phone" }
		},
		"required": ["x_calling_code", "x_national_number"]
	},
	{
		"properties": {
			"x_login_id_input_type": { "type": "string", "const": "text" }
		},
		"required": ["x_login_id"]
	}
	]
}
`

// nolint: gosec
const AuthorizeEnterPasswordRequestSchema = `
{
	"$id": "#AuthorizeEnterPasswordRequest",
	"type": "object",
	"properties": {
		"scope": { "type": "string", "const": "openid" },
		"response_type": { "type": "string", "const": "code" },
		"client_id": { "type": "string" },
		"redirect_uri": { "type": "string" },
		"code_challenge_method": { "type": "string", "const": "S256" },
		"code_challenge": { "type": "string" },
		"x_login_id_input_type": { "type": "string", "enum": ["phone", "text"] },
		"x_step": { "type": "string", "const": "submit_password" },
		"x_calling_code": { "type": "string" },
		"x_national_number": { "type": "string" },
		"x_login_id": { "type": "string" },
		"x_password": { "type": "string" }
	},
	"required": ["scope", "response_type", "client_id", "redirect_uri", "code_challenge_method", "code_challenge", "x_login_id_input_type", "x_step", "x_password"],
	"oneOf": [
	{
		"properties": {
			"x_login_id_input_type": { "type": "string", "const": "phone" }
		},
		"required": ["x_calling_code", "x_national_number"]
	},
	{
		"properties": {
			"x_login_id_input_type": { "type": "string", "const": "text" }
		},
		"required": ["x_login_id"]
	}
	]
}
`

func (h *AuthorizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		panic(err)
	}

	h.AuthContextProvider.Init(r)

	h.ValidateProvider.Prevalidate(r.Form)

	step := r.Form.Get("x_step")
	switch step {
	case "submit_password":
		t, data, err := h.SubmitPassword(w, r)
		h.RenderProvider.WritePage(w, r, t, data, err)
	case "submit_login_id":
		t, data, err := h.SubmitLoginID(w, r)
		h.RenderProvider.WritePage(w, r, t, data, err)
	default:
		t, data, err := h.Default(w, r)
		h.RenderProvider.WritePage(w, r, t, data, err)
	}
}

func (h *AuthorizeHandler) Default(w http.ResponseWriter, r *http.Request) (t config.TemplateItemType, data map[string]interface{}, err error) {
	t = template.TemplateItemTypeAuthUIAuthorizeHTML
	data = provider.FormToJSON(r.Form)
	err = h.ValidateProvider.Validate("#AuthorizeRequest", data)
	return
}

func (h *AuthorizeHandler) SubmitLoginID(w http.ResponseWriter, r *http.Request) (t config.TemplateItemType, data map[string]interface{}, err error) {
	t = template.TemplateItemTypeAuthUIAuthorizeHTML
	data = provider.FormToJSON(r.Form)
	err = h.ValidateProvider.Validate("#AuthorizeLoginIDRequest", data)
	if err == nil {
		t = template.TemplateItemTypeAuthUIEnterPasswordHTML
	}
	return
}

func (h *AuthorizeHandler) SubmitPassword(w http.ResponseWriter, r *http.Request) (t config.TemplateItemType, data map[string]interface{}, err error) {
	t = template.TemplateItemTypeAuthUIEnterPasswordHTML
	data = provider.FormToJSON(r.Form)
	err = h.ValidateProvider.Validate("#AuthorizeEnterPasswordRequest", data)
	if err != nil {
		return
	}

	loginID := provider.DeriveLoginID(r.Form)
	password := r.Form.Get("x_password")
	authnSession, err := h.AuthenticationProvider.AuthenticateWithPassword(loginID, password)
	if err != nil {
		return
	}

	// TODO(authui): Handle MFA
	if !authnSession.IsFinished() {
		panic("TODO(authui): Handle MFA")
	}

	// TODO(authui): Create session
	// TODO(authui): Dispatch hook
	// TODO(authui): Generate authorization code and redirect
	panic("TODO(authui)")
}
