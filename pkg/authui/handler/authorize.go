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
	})
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
	"if": {
		"properties": {
			"x_step": { "type": "string", "const": "input_login_id" }
		},
		"required": ["x_step"]
	},
	"then": {
		"oneOf": [
			{
				"properties": {
					"x_calling_code": { "type": "string" },
					"x_nation_number": { "type": "string", "minLength": 1 }
				},
				"required": ["x_calling_code", "x_national_number"]
			},
			{
				"properties": {
					"x_login_id": { "type": "string", "minLength": 1 }
				},
				"required": ["x_login_id"]
			}
		]
	},
	"required": ["scope", "response_type", "client_id", "redirect_uri", "code_challenge_method", "code_challenge"]
}
`

func (h *AuthorizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		// TODO(authui): Render error
		panic(err)
	}

	h.RenderProvider.PrevalidateForm(r.Form)

	switch r.Method {
	case "GET":
		data, _ := h.ValidateProvider.Validate("#AuthorizeRequest", r.Form)
		h.RenderProvider.WritePage(w, r, template.TemplateItemTypeAuthUIAuthorizeHTML, data)
	case "POST":
		break
	}
}
