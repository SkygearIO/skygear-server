package handler

import (
	"net/http"

	"github.com/gorilla/mux"

	coreTemplate "github.com/skygeario/skygear-server/pkg/core/template"

	"github.com/skygeario/skygear-server/pkg/authui/inject"
	"github.com/skygeario/skygear-server/pkg/authui/template"
)

type AuthorizeHandler struct {
	TemplateEngine *coreTemplate.Engine
}

func NewAuthorizeHandler(templateEngine *coreTemplate.Engine) *AuthorizeHandler {
	return &AuthorizeHandler{
		TemplateEngine: templateEngine,
	}
}

func AttachAuthorizeHandler(router *mux.Router, dep *inject.BootTimeDependency) {
	router.Path("/authorize").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		InjectAuthorizeHandler(r, dep).ServeHTTP(w, r)
	})
}

func (h *AuthorizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := h.TemplateEngine.RenderTemplate(
		template.TemplateItemTypeAuthUIAuthorizeHTML,
		map[string]interface{}{
			"logo_url":                  "https://via.placeholder.com/150",
			"skygear_logo_url":          "https://via.placeholder.com/65x15?text=Skygear",
			"x_login_id_type":           "phone",
			"x_login_id_type_has_phone": true,
			"x_login_id_type_has_text":  true,
		},
		coreTemplate.RenderOptions{},
		func(v *coreTemplate.Validator) {
			v.AllowRangeNode = true
		},
	)
	if err != nil {
		panic(err)
	}
	WriteHTML(w, body)
}
