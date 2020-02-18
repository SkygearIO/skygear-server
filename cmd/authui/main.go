package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"

	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/redis"
	"github.com/skygeario/skygear-server/pkg/core/sentry"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/validation"

	"github.com/skygeario/skygear-server/pkg/authui"
	"github.com/skygeario/skygear-server/pkg/authui/handler"
	"github.com/skygeario/skygear-server/pkg/authui/inject"
)

const ModuleName = "authui"

func main() {
	logging.SetModule(ModuleName)
	loggerFactory := logging.NewFactory(
		logging.NewDefaultLogHook(nil),
		&sentry.LogHook{Hub: sentry.DefaultClient.Hub},
	)
	logger := loggerFactory.NewLogger(ModuleName)

	if err := godotenv.Load(); err != nil {
		logger.WithError(err).Debug("cannot load .env file")
	}

	configuration := inject.Configuration{}
	envconfig.Process("", &configuration)

	var err error
	dbPool := db.NewPool()
	redisPool, err := redis.NewPool(configuration.Redis)
	if err != nil {
		logger.Fatalf("fail to create redis pool: %v", err.Error())
	}

	var standaloneConfig *config.TenantConfiguration
	if configuration.Standalone {
		filename := configuration.StandaloneTenantConfigurationFile
		reader, err := os.Open(filename)
		if err != nil {
			logger.WithError(err).Fatal("Cannot open standalone config")
		}
		tConfig, err := config.NewTenantConfigurationFromYAML(reader)
		if err != nil {
			logger.WithError(err).Fatal("Cannot parse standalone config")
		}
		standaloneConfig = tConfig
	}

	validator := validation.NewValidator("http://authui.skygear.io")
	validator.AddSchemaFragments(
		handler.AuthorizeRequestSchema,
		handler.AuthorizeLoginIDRequestSchema,
		handler.AuthorizeEnterPasswordRequestSchema,
	)

	dep := &inject.BootTimeDependency{
		Configuration:                 configuration,
		DBPool:                        dbPool,
		RedisPool:                     redisPool,
		StandaloneTenantConfiguration: standaloneConfig,
		Validator:                     validator,
	}
	router := authui.NewRouter(dep)

	handler.AttachRootHandler(router)
	handler.AttachAuthorizeHandler(router, dep)

	srv := &http.Server{
		Addr:    configuration.Host,
		Handler: router,
	}

	server.ListenAndServe(srv, logger, fmt.Sprintf("starting %s", ModuleName))
}
