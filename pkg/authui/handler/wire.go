//+build wireinject

package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/wire"

	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/passwordhistory"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal/password"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	redisSession "github.com/skygeario/skygear-server/pkg/core/auth/session/redis"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
	"github.com/skygeario/skygear-server/pkg/core/sentry"
	coreTemplate "github.com/skygeario/skygear-server/pkg/core/template"
	coreTime "github.com/skygeario/skygear-server/pkg/core/time"
	"github.com/skygeario/skygear-server/pkg/core/validation"

	"github.com/skygeario/skygear-server/pkg/authui/inject"
	"github.com/skygeario/skygear-server/pkg/authui/provider"
	"github.com/skygeario/skygear-server/pkg/authui/template"
)

var SessionKey = redisSession.SessionKeyFunc(func(appID string, sessionID string) string {
	return fmt.Sprintf("%s:auth-ui:session:%s", appID, sessionID)
})

var SessionListKey = redisSession.SessionListKeyFunc(func(appID string, sessionID string) string {
	return fmt.Sprintf("%s:auth-ui:session-list:%s", appID, sessionID)
})

var EventStreamKey = redisSession.EventStreamKeyFunc(func(appID string, sessionID string) string {
	return fmt.Sprintf("%s:auth-ui:event:%s", appID, sessionID)
})

func ProvideTenantConfig(r *http.Request) *config.TenantConfiguration {
	return config.GetTenantConfig(r.Context())
}

func ProvideContext(r *http.Request) context.Context {
	// NOTE(louis): This context must be used to store or retrieve values.
	// It is only intended to provide deadline associated with the request.
	return r.Context()
}

func ProvideAssetGearLoader(dep *inject.BootTimeDependency) *coreTemplate.AssetGearLoader {
	configuration := dep.Configuration
	if configuration.Template.AssetGearEndpoint != "" && configuration.Template.AssetGearMasterKey != "" {
		return &coreTemplate.AssetGearLoader{
			AssetGearEndpoint:  configuration.Template.AssetGearEndpoint,
			AssetGearMasterKey: configuration.Template.AssetGearMasterKey,
		}
	}
	return nil
}

func ProvideEnableFileSystemTemplate(dep *inject.BootTimeDependency) inject.EnableFileSystemTemplate {
	return inject.EnableFileSystemTemplate(dep.Configuration.Template.EnableFileLoader)
}

func ProvideValidator(dep *inject.BootTimeDependency) *validation.Validator {
	return dep.Validator
}

func ProvideReservedNameChecker(dep *inject.BootTimeDependency) *loginid.ReservedNameChecker {
	return dep.ReservedNameChecker
}

func ProvideLoggingFactory(tConfig *config.TenantConfiguration, ctx context.Context, r *http.Request) *logging.FactoryImpl {
	logHook := logging.NewDefaultLogHook(tConfig.DefaultSensitiveLoggerValues())
	sentryHook := sentry.NewLogHookFromContext(ctx)
	return logging.NewFactoryFromRequest(r, logHook, sentryHook)
}

func ProvideSessionStore(
	tConfig *config.TenantConfiguration,
	ctx context.Context,
	timeProvider coreTime.Provider,
	loggingFactory logging.Factory,
) *redisSession.StoreImpl {
	return redisSession.NewStore(
		ctx,
		tConfig.AppID,
		timeProvider,
		loggingFactory,
		SessionKey,
		SessionListKey,
	)
}

func ProvideSessionEventStore(
	tConfig *config.TenantConfiguration,
	ctx context.Context,
) *redisSession.EventStoreImpl {
	return redisSession.NewEventStore(
		ctx,
		tConfig.AppID,
		EventStreamKey,
	)
}

func ProvideSessionProvider(
	r *http.Request,
	tConfig *config.TenantConfiguration,
	store session.Store,
	eventStore session.EventStore,
	authContext coreAuth.ContextGetter,
	timeProvider coreTime.Provider,
) *session.ProviderImpl {
	return session.NewProvider(
		r,
		store,
		eventStore,
		authContext,
		tConfig.AppConfig.Clients,
	)
}

func ProvideSQLBuilder(tConfig *config.TenantConfiguration) db.SQLBuilder {
	return db.NewSQLBuilder("authui", tConfig.DatabaseConfig.DatabaseSchema, tConfig.AppID)
}

func ProvideSQLExecutor(ctx context.Context, dbContext db.Context) db.SQLExecutor {
	return db.NewSQLExecutor(ctx, dbContext)
}

func ProvidePasswordAuthProvider(
	store password.Store,
	historyStore passwordhistory.Store,
	loggerFactory logging.Factory,
	tConfig *config.TenantConfiguration,
	reservedNameChecker *loginid.ReservedNameChecker,
) *password.ProviderImpl {
	return password.NewProvider(
		store,
		historyStore,
		loggerFactory,
		tConfig.AppConfig.Auth.LoginIDKeys,
		tConfig.AppConfig.Auth.LoginIDTypes,
		tConfig.AppConfig.Auth.AllowedRealms,
		tConfig.AppConfig.PasswordPolicy.IsPasswordHistoryEnabled(),
		reservedNameChecker,
	)
}

var DefaultSet = wire.NewSet(
	ProvideTenantConfig,
	ProvideContext,
	ProvideAssetGearLoader,
	ProvideEnableFileSystemTemplate,
	ProvideValidator,
	ProvideReservedNameChecker,
	ProvideSQLBuilder,
	ProvideSQLExecutor,

	template.NewEngine,

	wire.Bind(new(coreTime.Provider), new(coreTime.ProviderImpl)),
	coreTime.NewProvider,

	wire.Bind(new(provider.RenderProvider), new(*provider.RenderProviderImpl)),
	provider.NewRenderProvider,

	wire.Bind(new(provider.ValidateProvider), new(*provider.ValidateProviderImpl)),
	provider.NewValidateProvider,

	wire.Bind(new(coreAuth.ContextGetter), new(*provider.AuthContextProviderImpl)),
	wire.Bind(new(provider.AuthContextProvider), new(*provider.AuthContextProviderImpl)),
	provider.NewAuthContextProvider,

	wire.Bind(new(logging.Factory), new(*logging.FactoryImpl)),
	ProvideLoggingFactory,

	wire.Bind(new(session.Store), new(*redisSession.StoreImpl)),
	ProvideSessionStore,
	wire.Bind(new(session.EventStore), new(*redisSession.EventStoreImpl)),
	ProvideSessionEventStore,
	wire.Bind(new(session.Provider), new(*session.ProviderImpl)),
	ProvideSessionProvider,

	wire.Bind(new(db.Context), new(*db.ContextImpl)),
	wire.Bind(new(db.TxContext), new(*db.ContextImpl)),
	wire.Bind(new(db.SafeTxContext), new(*db.ContextImpl)),
	db.NewContextImpl,

	wire.Bind(new(password.Store), new(*password.StoreImpl)),
	password.NewStore,

	wire.Bind(new(passwordhistory.Store), new(*passwordhistory.StoreImpl)),
	passwordhistory.NewPasswordHistoryStore,

	wire.Bind(new(password.Provider), new(*password.ProviderImpl)),
	ProvidePasswordAuthProvider,
)

func InjectRootHandler(r *http.Request) *RootHandler {
	wire.Build(NewRootHandler)
	return &RootHandler{}
}

func InjectAuthorizeHandler(r *http.Request, dep *inject.BootTimeDependency) *AuthorizeHandler {
	wire.Build(DefaultSet, NewAuthorizeHandler)
	return &AuthorizeHandler{}
}
