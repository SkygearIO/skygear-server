package handler

import (
	"net/http"

	"github.com/gorilla/mux"

	coreTemplate "github.com/skygeario/skygear-server/pkg/core/template"

	"github.com/skygeario/skygear-server/pkg/authui/inject"
	"github.com/skygeario/skygear-server/pkg/authui/provider"
	"github.com/skygeario/skygear-server/pkg/authui/template"
)

type AuthorizeHandler struct {
	TemplateEngine   *coreTemplate.Engine
	ValidateProvider provider.ValidateProvider
	RenderProvider   provider.RenderProvider
}

func NewAuthorizeHandler(templateEngine *coreTemplate.Engine, validateProvider provider.ValidateProvider, renderProvider provider.RenderProvider) *AuthorizeHandler {
	return &AuthorizeHandler{
		TemplateEngine:   templateEngine,
		ValidateProvider: validateProvider,
		RenderProvider:   renderProvider,
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
	"required": ["scope", "response_type", "client_id", "redirect_uri", "code_challenge_method", "code_challenge"],
	"dependencies": {
		"x_step": {
			"if": {
				"properties": {
					"x_step": { "type": "string", "const": "submit_login_id" }
				}
			},
			"then": {
				"oneOf": [
					{
						"properties": {
							"x_login_id_input_type": { "type": "string", "const": "phone" },
							"x_calling_code": { "type": "string", "minLength": 1 },
							"x_national_number": { "type": "string", "minLength": 1 }
						},
						"required": ["x_login_id_input_type", "x_calling_code", "x_national_number"]
					},
					{
						"properties": {
							"x_login_id_input_type": { "type": "string", "const": "text" },
							"x_login_id": { "type": "string", "minLength": 1 }
						},
						"required": ["x_login_id_input_type", "x_login_id"]
					}
				]
			}
		}
	}
}
`

func (h *AuthorizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		// TODO(authui): Render error
		panic(err)
	}

	h.ValidateProvider.Prevalidate(r.Form)

	step := r.Form.Get("x_step")

	switch step {
	case "submit_login_id":
		// TODO(authui): password page
		// If error, stay on the authorize page and display error.
		// Otherwise, render the password page.
		data, err := h.ValidateProvider.Validate("#AuthorizeRequest", r.Form)
		h.RenderProvider.WritePage(w, r, template.TemplateItemTypeAuthUIAuthorizeHTML, data, err)
	default:
		// Initial step: serve the authorize page
		data, err := h.ValidateProvider.Validate("#AuthorizeRequest", r.Form)
		h.RenderProvider.WritePage(w, r, template.TemplateItemTypeAuthUIAuthorizeHTML, data, err)
	}

}
