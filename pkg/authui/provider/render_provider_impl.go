package provider

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/phone"
	"github.com/skygeario/skygear-server/pkg/core/template"
)

type RenderProviderImpl struct {
	AppName        string
	TemplateEngine *template.Engine
	LoginIDKeys    []config.LoginIDKeyConfiguration
}

var _ RenderProvider = &RenderProviderImpl{}

func NewRenderProvider(tConfig *config.TenantConfiguration, templateEngine *template.Engine) *RenderProviderImpl {
	return &RenderProviderImpl{
		AppName:        tConfig.AppConfig.DisplayAppName,
		TemplateEngine: templateEngine,
		LoginIDKeys:    tConfig.AppConfig.Auth.LoginIDKeys,
	}
}

func (p *RenderProviderImpl) PrevalidateForm(form url.Values) {
	if _, ok := form["x_login_id_input_type"]; !ok {
		if len(p.LoginIDKeys) > 0 {
			if string(p.LoginIDKeys[0].Type) == "phone" {
				form.Set("x_login_id_input_type", "phone")
			} else {
				form.Set("x_login_id_input_type", "text")
			}
		}
	}
}

func (p *RenderProviderImpl) WritePage(
	w http.ResponseWriter,
	r *http.Request,
	templateType config.TemplateItemType,
	data map[string]interface{},
) {
	data["appname"] = p.AppName

	// TODO(authui): configure logo URL
	data["logo_url"] = "https://via.placeholder.com/150"

	// TODO(authui): asset skygear logo URL
	data["skygear_logo_url"] = "https://via.placeholder.com/65x15?text=Skygear"

	data["x_calling_codes"] = phone.CountryCallingCodes

	for _, keyConfig := range p.LoginIDKeys {
		if string(keyConfig.Type) == "phone" {
			data["x_login_id_input_type_has_phone"] = true
		} else {
			data["x_login_id_input_type_has_text"] = true
		}
	}

	r.Form.Set("x_login_id_input_type", "phone")
	data["x_use_phone_url"] = fmt.Sprintf("?%s", r.Form.Encode())

	r.Form.Set("x_login_id_input_type", "text")
	data["x_use_text_url"] = fmt.Sprintf("?%s", r.Form.Encode())

	out, err := p.TemplateEngine.RenderTemplate(templateType, data, template.RenderOptions{}, func(v *template.Validator) {
		v.AllowRangeNode = true
	})
	if err != nil {
		panic(err)
	}
	body := []byte(out)
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(200)
	w.Write(body)
}
