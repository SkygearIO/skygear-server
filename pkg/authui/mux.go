package authui

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/middleware"
	"github.com/skygeario/skygear-server/pkg/core/sentry"
	"github.com/skygeario/skygear-server/pkg/core/server"

	"github.com/skygeario/skygear-server/pkg/authui/inject"
)

func InjectTenantConfigMiddleware(tenantConfig *config.TenantConfiguration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(config.WithTenantConfig(r.Context(), tenantConfig))
			next.ServeHTTP(w, r)
		})
	}
}

func NewRouter(dep *inject.BootTimeDependency) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/healthz", server.HealthCheckHandler)

	router.Use(sentry.Middleware(sentry.DefaultClient.Hub))
	router.Use(middleware.RecoverMiddleware{}.Handle)

	if dep.StandaloneTenantConfiguration != nil {
		router.Use(InjectTenantConfigMiddleware(dep.StandaloneTenantConfiguration))
		router.Use(middleware.RequestIDMiddleware{}.Handle)
		router.Use(middleware.CORSMiddleware{}.Handle)
	} else {
		router.Use(middleware.ReadTenantConfigMiddleware{}.Handle)
	}

	router.Use(middleware.DBMiddleware{Pool: dep.DBPool}.Handle)
	router.Use(middleware.RedisMiddleware{Pool: dep.RedisPool}.Handle)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentSecurityPolicy := "frame-ancestors 'self'"
			tConfig := config.GetTenantConfig(r.Context())
			if tConfig.AppConfig.AuthUI.ContentSecurityPolicy != "" {
				contentSecurityPolicy = tConfig.AppConfig.AuthUI.ContentSecurityPolicy
			}
			w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
			next.ServeHTTP(w, r)
		})
	})

	if dep.Configuration.StaticAssetDir != "" {
		router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(dep.Configuration.StaticAssetDir))))
	}

	return router
}
