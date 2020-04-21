package auth

import (
	"context"
	"net/http"

	authAudit "github.com/skygeario/skygear-server/pkg/auth/dependency/audit"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/loginid"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/mfa"
	mfaPQ "github.com/skygeario/skygear-server/pkg/auth/dependency/mfa/pq"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/passwordhistory"
	pqPWHistory "github.com/skygeario/skygear-server/pkg/auth/dependency/passwordhistory/pq"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/oauth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/password"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/sso"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/urlprefix"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userverify"
	authTemplate "github.com/skygeario/skygear-server/pkg/auth/template"
	"github.com/skygeario/skygear-server/pkg/core/async"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	pqAuthInfo "github.com/skygeario/skygear-server/pkg/core/auth/authinfo/pq"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/mail"
	"github.com/skygeario/skygear-server/pkg/core/sentry"
	"github.com/skygeario/skygear-server/pkg/core/sms"
	"github.com/skygeario/skygear-server/pkg/core/template"
	"github.com/skygeario/skygear-server/pkg/core/time"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

type DependencyMap struct {
	EnableFileSystemTemplate bool
	Validator                *validation.Validator
	AssetGearLoader          *template.AssetGearLoader
	AsyncTaskExecutor        *async.Executor
	UseInsecureCookie        bool
	StaticAssetURLPrefix     string
	DefaultConfiguration     config.DefaultConfiguration
	ReservedNameChecker      *loginid.ReservedNameChecker
}

// Provide provides dependency instance by name
// nolint: gocyclo, golint
func (m DependencyMap) Provide(
	dependencyName string,
	request *http.Request,
	ctx context.Context,
	requestID string,
	tc config.TenantConfiguration,
) interface{} {
	// populate default
	appConfig := tc.AppConfig
	if !appConfig.SMTP.IsValid() {
		appConfig.SMTP = m.DefaultConfiguration.SMTP
	}

	if !appConfig.Twilio.IsValid() {
		appConfig.Twilio = m.DefaultConfiguration.Twilio
	}

	if !appConfig.Nexmo.IsValid() {
		appConfig.Nexmo = m.DefaultConfiguration.Nexmo
	}

	// To avoid mutating tc
	tConfig := tc
	tConfig.AppConfig = appConfig

	newLoggerFactory := func() logging.Factory {
		logHook := logging.NewDefaultLogHook(tConfig.DefaultSensitiveLoggerValues())
		sentryHook := sentry.NewLogHookFromContext(ctx)
		if request == nil {
			return logging.NewFactoryFromRequestID(requestID, logHook, sentryHook)
		} else {
			return logging.NewFactoryFromRequest(request, logHook, sentryHook)
		}
	}

	newSQLBuilder := func() db.SQLBuilder {
		return db.NewSQLBuilder("auth", tConfig.DatabaseConfig.DatabaseSchema, tConfig.AppID)
	}

	newSQLExecutor := func() db.SQLExecutor {
		return db.NewSQLExecutor(ctx, db.NewContextWithContext(ctx, tConfig))
	}

	newTimeProvider := func() time.Provider {
		return time.NewProvider()
	}

	newPasswordStore := func() password.Store {
		return password.NewStore(
			newSQLBuilder(),
			newSQLExecutor(),
		)
	}

	newPasswordHistoryStore := func() passwordhistory.Store {
		return pqPWHistory.NewPasswordHistoryStore(
			newTimeProvider(),
			newSQLBuilder(),
			newSQLExecutor(),
		)
	}

	newTemplateEngine := func() *template.Engine {
		return authTemplate.NewEngineWithConfig(
			tConfig,
			m.EnableFileSystemTemplate,
			m.AssetGearLoader,
		)
	}

	newAuthInfoStore := func() authinfo.Store {
		return pqAuthInfo.NewAuthInfoStore(
			db.NewSQLBuilder("core", tConfig.DatabaseConfig.DatabaseSchema, tConfig.AppID),
			newSQLExecutor(),
		)
	}

	newUserProfileStore := func() userprofile.Store {
		return userprofile.NewUserProfileStore(
			newTimeProvider(),
			newSQLBuilder(),
			newSQLExecutor(),
		)
	}

	newLoginIDChecker := func() loginid.LoginIDChecker {
		return loginid.NewDefaultLoginIDChecker(
			tConfig.AppConfig.Identity.LoginID.Keys,
			tConfig.AppConfig.Identity.LoginID.Types,
			m.ReservedNameChecker,
		)
	}

	newPasswordAuthProvider := func() password.Provider {
		return password.NewProvider(
			newTimeProvider(),
			newPasswordStore(),
			newPasswordHistoryStore(),
			newLoggerFactory(),
			tConfig.AppConfig.Identity.LoginID.Keys,
			tConfig.AppConfig.Identity.LoginID.Types,
			tConfig.AppConfig.Authenticator.Password.Policy.IsPasswordHistoryEnabled(),
			m.ReservedNameChecker,
		)
	}

	newOAuthAuthProvider := func() oauth.Provider {
		return oauth.NewProvider(
			newSQLBuilder(),
			newSQLExecutor(),
		)
	}

	newHookProvider := func() hook.Provider {
		return inject.Scoped(ctx, "HookProvider", func() interface{} {
			return hook.NewProvider(
				ctx,
				requestID,
				hook.NewStore(newSQLBuilder(), newSQLExecutor()),
				db.NewTxContextWithContext(ctx, tConfig),
				newTimeProvider(),
				newAuthInfoStore(),
				newUserProfileStore(),
				hook.NewDeliverer(
					&tConfig,
					newTimeProvider(),
					hook.NewMutator(
						tConfig.AppConfig.UserVerification,
						newPasswordAuthProvider(),
						newAuthInfoStore(),
						newUserProfileStore(),
					),
				),
				newLoggerFactory(),
			)
		})().(hook.Provider)
	}

	newIdentityProvider := func() principal.IdentityProvider {
		return principal.NewIdentityProvider(
			newSQLBuilder(),
			newSQLExecutor(),
			newOAuthAuthProvider(),
			newPasswordAuthProvider(),
		)
	}

	newSMSClient := func() sms.Client {
		return sms.NewClient(ctx, tConfig.AppConfig)
	}

	newMailSender := func() mail.Sender {
		return mail.NewSender(ctx, tConfig.AppConfig.SMTP)
	}

	newMFAProvider := func() mfa.Provider {
		return mfa.NewProvider(
			mfaPQ.NewStore(
				tConfig.AppConfig.Authenticator.RecoveryCode,
				newSQLBuilder(),
				newSQLExecutor(),
				newTimeProvider(),
			),
			tConfig.AppConfig.Authenticator,
			newTimeProvider(),
			mfa.NewSender(
				tConfig,
				newSMSClient(),
				newMailSender(),
				newTemplateEngine(),
			),
		)
	}

	newLoginIDNormalizerFactory := func() loginid.LoginIDNormalizerFactory {
		return loginid.NewLoginIDNormalizerFactory(
			tConfig.AppConfig.Identity.LoginID.Keys,
			tConfig.AppConfig.Identity.LoginID.Types,
		)
	}

	newPasswordChecker := func() *authAudit.PasswordChecker {
		return authAudit.ProvidePasswordChecker(&tConfig, newPasswordHistoryStore())
	}

	switch dependencyName {
	case "TxContext":
		return db.NewTxContextWithContext(ctx, tConfig)
	case "LoggerFactory":
		return newLoggerFactory()
	case "RequireAuthz":
		return handler.NewRequireAuthzFactory(newLoggerFactory())
	case "Validator":
		return m.Validator
	case "MFAProvider":
		return newMFAProvider()
	case "AuthInfoStore":
		return newAuthInfoStore()
	case "PasswordChecker":
		return newPasswordChecker()
	case "LoginIDChecker":
		return newLoginIDChecker()
	case "PasswordAuthProvider":
		return newPasswordAuthProvider()
	case "HandlerLogger":
		return newLoggerFactory().NewLogger("handler")
	case "UserProfileStore":
		return newUserProfileStore()
	case "ForgotPasswordSecureMatch":
		return tConfig.AppConfig.ForgotPassword.SecureMatch
	case "UserVerifyCodeSenderFactory":
		return userverify.NewDefaultUserVerifyCodeSenderFactory(
			&tConfig,
			newTemplateEngine(),
			newMailSender(),
			newSMSClient(),
		)
	case "AutoSendUserVerifyCodeOnSignup":
		return tConfig.AppConfig.UserVerification.AutoSendOnSignup
	case "UserVerifyLoginIDKeys":
		return tConfig.AppConfig.UserVerification.LoginIDKeys
	case "UserVerificationProvider":
		return userverify.NewProvider(
			userverify.NewCodeGenerator(&tConfig),
			userverify.NewStore(
				newSQLBuilder(),
				newSQLExecutor(),
			),
			tConfig.AppConfig.UserVerification,
			newTimeProvider(),
		)
	case "VerifyHTMLProvider":
		return userverify.NewVerifyHTMLProvider(tConfig.AppConfig.UserVerification, newTemplateEngine())
	case "LoginIDNormalizerFactory":
		return newLoginIDNormalizerFactory()
	case "IdentityProvider":
		return newIdentityProvider()
	case "AuthHandlerHTMLProvider":
		return sso.NewAuthHandlerHTMLProvider(urlprefix.NewProvider(request).Value())
	case "AsyncTaskQueue":
		return async.NewQueue(ctx, db.NewTxContextWithContext(ctx, tConfig), requestID, &tConfig, m.AsyncTaskExecutor)
	case "HookProvider":
		return newHookProvider()
	case "AuthenticatorConfiguration":
		return *tConfig.AppConfig.Authenticator
	case "OAuthConflictConfiguration":
		return tConfig.AppConfig.AuthAPI.OnIdentityConflict.OAuth
	case "TenantConfiguration":
		return &tConfig
	case "URLPrefix":
		return urlprefix.NewProvider(request).Value()
	case "TemplateEngine":
		return newTemplateEngine()
	case "TimeProvider":
		return newTimeProvider()
	case "SessionManager":
		return newSessionManager(request, m)
	default:
		return nil
	}
}
