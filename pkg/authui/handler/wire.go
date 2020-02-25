//+build wireinject

package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/wire"

	"github.com/skygeario/skygear-server/pkg/core/audit"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	authinfopq "github.com/skygeario/skygear-server/pkg/core/auth/authinfo/pq"
	"github.com/skygeario/skygear-server/pkg/core/auth/mfa"
	"github.com/skygeario/skygear-server/pkg/core/auth/passwordhistory"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal/password"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	redisSession "github.com/skygeario/skygear-server/pkg/core/auth/session/redis"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
	"github.com/skygeario/skygear-server/pkg/core/mail"
	"github.com/skygeario/skygear-server/pkg/core/sentry"
	"github.com/skygeario/skygear-server/pkg/core/sms"
	coreTemplate "github.com/skygeario/skygear-server/pkg/core/template"
	coreTime "github.com/skygeario/skygear-server/pkg/core/time"
	"github.com/skygeario/skygear-server/pkg/core/urlprefix"
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

func ProvideTenantConfigPtr(r *http.Request) *config.TenantConfiguration {
	return config.GetTenantConfig(r.Context())
}

func ProvideTenantConfig(r *http.Request) config.TenantConfiguration {
	return *config.GetTenantConfig(r.Context())
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
	// We still have to reference _auth_* tables.
	return db.NewSQLBuilder("auth", tConfig.DatabaseConfig.DatabaseSchema, tConfig.AppID)
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

func ProvideAuditTrail(tConfig *config.TenantConfiguration) audit.Trail {
	t, err := audit.NewTrail(tConfig.AppConfig.UserAudit.Enabled, tConfig.AppConfig.UserAudit.TrailHandlerURL)
	if err != nil {
		panic(err)
	}
	return t
}

func ProvideSMSClient(tConfig *config.TenantConfiguration) *sms.ClientImpl {
	return sms.NewClient(tConfig.AppConfig)
}

func ProvideMailSender(tConfig *config.TenantConfiguration) *mail.SenderImpl {
	return mail.NewSender(tConfig.AppConfig.SMTP)
}

func ProvideMFAStore(
	tConfig *config.TenantConfiguration,
	sqlBuilder db.SQLBuilder,
	sqlExecutor db.SQLExecutor,
	timeProvider coreTime.Provider,
) *mfa.StoreImpl {
	return mfa.NewStore(tConfig.AppConfig.MFA, sqlBuilder, sqlExecutor, timeProvider)
}

func ProvideMFAProvider(
	store mfa.Store,
	tConfig *config.TenantConfiguration,
	timeProvider coreTime.Provider,
	sender mfa.Sender,
) *mfa.ProviderImpl {
	return mfa.NewProvider(store, tConfig.AppConfig.MFA, timeProvider, sender)
}

var DefaultSet = wire.NewSet(
	ProvideTenantConfig,
	ProvideTenantConfigPtr,
	ProvideContext,
	ProvideAssetGearLoader,
	ProvideEnableFileSystemTemplate,
	ProvideValidator,
	ProvideReservedNameChecker,
	ProvideSQLBuilder,
	ProvideSQLExecutor,

	template.NewEngine,

	urlprefix.NewProvider,

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

	wire.Bind(new(authinfo.Store), new(*authinfopq.StoreImpl)),
	authinfopq.NewAuthInfoStore,

	wire.Bind(new(userprofile.Store), new(*userprofile.StoreImpl)),
	userprofile.NewUserProfileStore,

	wire.Bind(new(password.Store), new(*password.StoreImpl)),
	password.NewStore,

	wire.Bind(new(passwordhistory.Store), new(*passwordhistory.StoreImpl)),
	passwordhistory.NewPasswordHistoryStore,

	wire.Bind(new(password.Provider), new(*password.ProviderImpl)),
	ProvidePasswordAuthProvider,

	ProvideAuditTrail,

	wire.Bind(new(sms.Client), new(*sms.ClientImpl)),
	ProvideSMSClient,

	wire.Bind(new(mail.Sender), new(*mail.SenderImpl)),
	ProvideMailSender,

	wire.Bind(new(mfa.Sender), new(*mfa.SenderImpl)),
	mfa.NewSender,
	wire.Bind(new(mfa.Store), new(*mfa.StoreImpl)),
	ProvideMFAStore,
	wire.Bind(new(mfa.Provider), new(*mfa.ProviderImpl)),
	ProvideMFAProvider,

	wire.Bind(new(provider.AuthenticationProvider), new(*provider.AuthenticationProviderImpl)),
	provider.NewAuthenticationProvider,
)

func InjectRootHandler(r *http.Request) *RootHandler {
	wire.Build(NewRootHandler)
	return &RootHandler{}
}

func InjectAuthorizeHandler(r *http.Request, dep *inject.BootTimeDependency) *AuthorizeHandler {
	wire.Build(DefaultSet, NewAuthorizeHandler)
	return &AuthorizeHandler{}
}
