package auth

import (
	"context"
	"net/http"

	"github.com/google/wire"
	"github.com/gorilla/mux"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/audit"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/auth"
	authredis "github.com/skygeario/skygear-server/pkg/auth/dependency/auth/redis"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authn"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/forgotpassword"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/loginid"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/mfa"
	mfapq "github.com/skygeario/skygear-server/pkg/auth/dependency/mfa/pq"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/oauth"
	oauthhandler "github.com/skygeario/skygear-server/pkg/auth/dependency/oauth/handler"
	oauthpq "github.com/skygeario/skygear-server/pkg/auth/dependency/oauth/pq"
	oauthredis "github.com/skygeario/skygear-server/pkg/auth/dependency/oauth/redis"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/oidc"
	oidchandler "github.com/skygeario/skygear-server/pkg/auth/dependency/oidc/handler"
	passwordhistorypq "github.com/skygeario/skygear-server/pkg/auth/dependency/passwordhistory/pq"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal"
	oauthprincipal "github.com/skygeario/skygear-server/pkg/auth/dependency/principal/oauth"
	passwordprincipal "github.com/skygeario/skygear-server/pkg/auth/dependency/principal/password"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/session"
	sessionredis "github.com/skygeario/skygear-server/pkg/auth/dependency/session/redis"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/sso"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/urlprefix"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userverify"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/webapp"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/welcemail"
	"github.com/skygeario/skygear-server/pkg/auth/template"
	"github.com/skygeario/skygear-server/pkg/core/async"
	coreauth "github.com/skygeario/skygear-server/pkg/core/auth"
	authinfopq "github.com/skygeario/skygear-server/pkg/core/auth/authinfo/pq"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	corehttp "github.com/skygeario/skygear-server/pkg/core/http"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/mail"
	"github.com/skygeario/skygear-server/pkg/core/sms"
	coretemplate "github.com/skygeario/skygear-server/pkg/core/template"
	"github.com/skygeario/skygear-server/pkg/core/time"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

func MakeHandler(deps DependencyMap, factory func(r *http.Request, m DependencyMap) http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := factory(r, deps)
		h.ServeHTTP(w, r)
	})
}

func MakeMiddleware(deps DependencyMap, factory func(r *http.Request, m DependencyMap) mux.MiddlewareFunc) mux.MiddlewareFunc {
	return mux.MiddlewareFunc(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			m := factory(r, deps)
			h := m(next)
			h.ServeHTTP(w, r)
		})
	})
}

func ProvideContext(r *http.Request) context.Context {
	return r.Context()
}

func ProvideLoggingRequestID(r *http.Request) logging.RequestID {
	return logging.RequestID(r.Header.Get(corehttp.HeaderRequestID))
}

func ProvideTenantConfig(ctx context.Context) *config.TenantConfiguration {
	return config.GetTenantConfig(ctx)
}

func ProvideSessionInsecureCookieConfig(m DependencyMap) session.InsecureCookieConfig {
	return session.InsecureCookieConfig(m.UseInsecureCookie)
}

func ProvideMFAInsecureCookieConfig(m DependencyMap) mfa.InsecureCookieConfig {
	return mfa.InsecureCookieConfig(m.UseInsecureCookie)
}

func ProvideValidator(m DependencyMap) *validation.Validator {
	return m.Validator
}

func ProvideReservedNameChecker(m DependencyMap) *loginid.ReservedNameChecker {
	return m.ReservedNameChecker
}

func ProvideTaskExecutor(m DependencyMap) *async.Executor {
	return m.AsyncTaskExecutor
}

func ProvideTemplateEngine(config *config.TenantConfiguration, m DependencyMap) *coretemplate.Engine {
	return template.NewEngineWithConfig(
		*config,
		m.EnableFileSystemTemplate,
		m.AssetGearLoader,
	)
}

func ProvideAuthSQLBuilder(f db.SQLBuilderFactory) db.SQLBuilder {
	return f("auth")
}

func ProvidePrincipalProviders(
	oauth oauthprincipal.Provider,
	password passwordprincipal.Provider,
) []principal.Provider {
	return []principal.Provider{oauth, password}
}

// ProvideWebAppRenderProvider is placed here because it requires DependencyMap.
func ProvideWebAppRenderProvider(
	m DependencyMap,
	config *config.TenantConfiguration,
	templateEngine *coretemplate.Engine,
	passwordChecker *audit.PasswordChecker,
) webapp.RenderProvider {
	return &webapp.RenderProviderImpl{
		StaticAssetURLPrefix:        m.StaticAssetURLPrefix,
		IdentityConfiguration:       config.AppConfig.Identity,
		AuthenticationConfiguration: config.AppConfig.Authentication,
		AuthUIConfiguration:         config.AppConfig.AuthUI,
		PasswordChecker:             passwordChecker,
		TemplateEngine:              templateEngine,
	}
}

func ProvideCSRFMiddleware(m DependencyMap, tConfig *config.TenantConfiguration) mux.MiddlewareFunc {
	middleware := &webapp.CSRFMiddleware{
		// NOTE(webapp): reuse Authentication.Secret instead of creating a new one.
		Key:               tConfig.AppConfig.Authentication.Secret,
		UseInsecureCookie: m.UseInsecureCookie,
	}
	return middleware.Handle
}

var CommonDependencySet = wire.NewSet(
	ProvideTenantConfig,
	ProvideSessionInsecureCookieConfig,
	ProvideMFAInsecureCookieConfig,
	ProvideValidator,
	ProvideReservedNameChecker,
	ProvideTaskExecutor,
	ProvideTemplateEngine,
	ProvideWebAppRenderProvider,
	endpointsProviderSet,

	ProvideAuthSQLBuilder,
	ProvidePrincipalProviders,

	logging.DependencySet,
	time.DependencySet,
	db.DependencySet,
	authinfopq.DependencySet,
	userprofile.DependencySet,
	session.DependencySet,
	sessionredis.DependencySet,
	handler.DependencySet,
	coreauth.DependencySet,
	async.DependencySet,
	sms.DependencySet,
	mail.DependencySet,

	hook.DependencySet,
	auth.DependencySet,
	authredis.DependencySet,
	authn.DependencySet,
	audit.DependencySet,
	loginid.DependencySet,
	passwordhistorypq.DependencySet,
	principal.DependencySet,
	oauthprincipal.DependencySet,
	passwordprincipal.DependencySet,
	sso.DependencySet,
	urlprefix.DependencySet,
	mfa.DependencySet,
	mfapq.DependencySet,
	webapp.DependencySet,
	oauthhandler.DependencySet,
	oauth.DependencySet,
	oauthpq.DependencySet,
	oauthredis.DependencySet,
	oidc.DependencySet,
	oidchandler.DependencySet,
	welcemail.DependencySet,
	userverify.DependencySet,
	forgotpassword.DependencySet,
)

// DependencySet is for HTTP request
var DependencySet = wire.NewSet(
	CommonDependencySet,
	ProvideContext,
	ProvideLoggingRequestID,
)
