package provider

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/core/config"
)

type RenderProvider interface {
	WritePage(w http.ResponseWriter, r *http.Request, templateType config.TemplateItemType, data map[string]interface{}, err error)
}
