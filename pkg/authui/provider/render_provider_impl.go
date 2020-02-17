package provider

import (
	"net/http"
	"strconv"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/template"
)

type RenderProviderImpl struct {
	AppName        string
	TemplateEngine *template.Engine
}

var _ RenderProvider = &RenderProviderImpl{}

func NewRenderProvider(tConfig *config.TenantConfiguration, templateEngine *template.Engine) *RenderProviderImpl {
	return &RenderProviderImpl{
		AppName:        tConfig.AppConfig.DisplayAppName,
		TemplateEngine: templateEngine,
	}
}

func (p *RenderProviderImpl) WritePage(w http.ResponseWriter, templateType config.TemplateItemType, data map[string]interface{}) {
	data["appname"] = p.AppName
	data["logo_url"] = "https://via.placeholder.com/150"
	data["skygear_logo_url"] = "https://via.placeholder.com/65x15?text=Skygear"
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
