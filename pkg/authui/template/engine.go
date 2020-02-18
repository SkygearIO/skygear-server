package template

import (
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/template"

	"github.com/skygeario/skygear-server/pkg/authui/inject"
)

func NewEngine(tConfig *config.TenantConfiguration, fs inject.EnableFileSystemTemplate, assetGearLoader *template.AssetGearLoader) *template.Engine {
	e := template.NewEngine(template.NewEngineOptions{
		EnableFileLoader: bool(fs),
		TemplateItems:    tConfig.TemplateItems,
		AssetGearLoader:  assetGearLoader,
	})

	e.Register(TemplateAuthUIAuthorizeHTML)
	e.Register(TemplateAuthUIEnterPasswordHTML)

	return e
}
