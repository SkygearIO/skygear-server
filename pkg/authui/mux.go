package authui

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/gomodule/redigo/redis"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/middleware"
	"github.com/skygeario/skygear-server/pkg/core/sentry"
	"github.com/skygeario/skygear-server/pkg/core/server"

	"github.com/skygeario/skygear-server/pkg/authui/handler"
)

func InjectTenantConfigMiddleware(tenantConfig *config.TenantConfiguration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r = r.WithContext(config.WithTenantConfig(r.Context(), tenantConfig))
			next.ServeHTTP(w, r)
		})
	}
}

func NewRouter(dbPool db.Pool, redisPool *redis.Pool, tenantConfig *config.TenantConfiguration) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/healthz", server.HealthCheckHandler)

	router.Use(sentry.Middleware(sentry.DefaultClient.Hub))
	router.Use(middleware.RecoverMiddleware{}.Handle)

	if tenantConfig != nil {
		router.Use(InjectTenantConfigMiddleware(tenantConfig))
		router.Use(middleware.RequestIDMiddleware{}.Handle)
		router.Use(middleware.CORSMiddleware{}.Handle)
	} else {
		router.Use(middleware.ReadTenantConfigMiddleware{}.Handle)
	}

	router.Use(middleware.DBMiddleware{Pool: dbPool}.Handle)
	router.Use(middleware.RedisMiddleware{Pool: redisPool}.Handle)

	router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.InjectRootHandler(r).ServeHTTP(w, r)
	})

	return router
}
