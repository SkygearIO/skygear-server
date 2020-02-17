package provider

import (
	"net/http"
	"net/url"

	"github.com/skygeario/skygear-server/pkg/core/config"
)

type RenderProvider interface {
	PrevalidateForm(values url.Values)
	WritePage(w http.ResponseWriter, templateType config.TemplateItemType, data map[string]interface{})
}
