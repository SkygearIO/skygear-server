// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package handler

import (
	"github.com/skygeario/skygear-server/pkg/auth"
	auth2 "github.com/skygeario/skygear-server/pkg/auth/dependency/auth"
	redis3 "github.com/skygeario/skygear-server/pkg/auth/dependency/auth/redis"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authenticator/bearertoken"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authenticator/oob"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authenticator/password"
	provider2 "github.com/skygeario/skygear-server/pkg/auth/dependency/authenticator/provider"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authenticator/recoverycode"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/authenticator/totp"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/challenge"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/identity/anonymous"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/identity/loginid"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/identity/oauth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/identity/provider"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/interaction"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/interaction/flows"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/interaction/redis"
	oauth2 "github.com/skygeario/skygear-server/pkg/auth/dependency/oauth"
	handler2 "github.com/skygeario/skygear-server/pkg/auth/dependency/oauth/handler"
	pq2 "github.com/skygeario/skygear-server/pkg/auth/dependency/oauth/pq"
	redis2 "github.com/skygeario/skygear-server/pkg/auth/dependency/oauth/redis"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/oidc"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/session"
	redis4 "github.com/skygeario/skygear-server/pkg/auth/dependency/session/redis"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/urlprefix"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/welcomemessage"
	"github.com/skygeario/skygear-server/pkg/core/async"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo/pq"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/logging"
	"github.com/skygeario/skygear-server/pkg/core/time"
	"github.com/skygeario/skygear-server/pkg/core/validation"
	"net/http"
)

// Injectors from wire.go:

func newLoginHandler(r *http.Request, m auth.DependencyMap) http.Handler {
	context := auth.ProvideContext(r)
	requestID := auth.ProvideLoggingRequestID(r)
	tenantConfiguration := auth.ProvideTenantConfig(context, m)
	factory := logging.ProvideLoggerFactory(context, requestID, tenantConfiguration)
	requireAuthz := handler.NewRequireAuthzFactory(factory)
	validator := auth.ProvideValidator(m)
	timeProvider := time.NewProvider()
	store := redis.ProvideStore(context, tenantConfiguration, timeProvider)
	sqlBuilderFactory := db.ProvideSQLBuilderFactory(tenantConfiguration)
	sqlBuilder := auth.ProvideAuthSQLBuilder(sqlBuilderFactory)
	sqlExecutor := db.ProvideSQLExecutor(context, tenantConfiguration)
	reservedNameChecker := auth.ProvideReservedNameChecker(m)
	typeCheckerFactory := loginid.ProvideTypeCheckerFactory(tenantConfiguration, reservedNameChecker)
	checker := loginid.ProvideChecker(tenantConfiguration, typeCheckerFactory)
	normalizerFactory := loginid.ProvideNormalizerFactory(tenantConfiguration)
	loginidProvider := loginid.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration, checker, normalizerFactory)
	oauthProvider := oauth.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider)
	anonymousProvider := anonymous.ProvideProvider(sqlBuilder, sqlExecutor)
	providerProvider := provider.ProvideProvider(tenantConfiguration, loginidProvider, oauthProvider, anonymousProvider)
	historyStoreImpl := password.ProvideHistoryStore(timeProvider, sqlBuilder, sqlExecutor)
	passwordChecker := password.ProvideChecker(tenantConfiguration, historyStoreImpl)
	passwordProvider := password.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, factory, historyStoreImpl, passwordChecker, tenantConfiguration)
	totpProvider := totp.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	engine := auth.ProvideTemplateEngine(tenantConfiguration, m)
	urlprefixProvider := urlprefix.NewProvider(r)
	txContext := db.ProvideTxContext(context, tenantConfiguration)
	executor := auth.ProvideTaskExecutor(m)
	queue := async.ProvideTaskQueue(context, txContext, requestID, tenantConfiguration, executor)
	oobProvider := oob.ProvideProvider(tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider, engine, urlprefixProvider, queue)
	bearertokenProvider := bearertoken.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	recoverycodeProvider := recoverycode.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	provider3 := &provider2.Provider{
		Password:     passwordProvider,
		TOTP:         totpProvider,
		OOBOTP:       oobProvider,
		BearerToken:  bearertokenProvider,
		RecoveryCode: recoverycodeProvider,
	}
	authinfoStore := pq.ProvideStore(sqlBuilderFactory, sqlExecutor)
	userprofileStore := userprofile.ProvideStore(timeProvider, sqlBuilder, sqlExecutor)
	hookProvider := hook.ProvideHookProvider(context, sqlBuilder, sqlExecutor, requestID, tenantConfiguration, txContext, timeProvider, authinfoStore, userprofileStore, loginidProvider, factory)
	welcomemessageProvider := welcomemessage.ProvideProvider(tenantConfiguration, engine, queue)
	userProvider := interaction.ProvideUserProvider(authinfoStore, userprofileStore, timeProvider, hookProvider, urlprefixProvider, queue, tenantConfiguration, welcomemessageProvider)
	interactionProvider := interaction.ProvideProvider(store, timeProvider, factory, providerProvider, provider3, userProvider, oobProvider, tenantConfiguration, hookProvider)
	authorizationStore := &pq2.AuthorizationStore{
		SQLBuilder:  sqlBuilder,
		SQLExecutor: sqlExecutor,
	}
	grantStore := redis2.ProvideGrantStore(context, factory, tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider)
	eventStore := redis3.ProvideEventStore(context, tenantConfiguration)
	accessEventProvider := auth2.AccessEventProvider{
		Store: eventStore,
	}
	sessionStore := redis4.ProvideStore(context, tenantConfiguration, timeProvider, factory)
	authAccessEventProvider := &auth2.AccessEventProvider{
		Store: eventStore,
	}
	sessionProvider := session.ProvideSessionProvider(r, sessionStore, authAccessEventProvider, tenantConfiguration)
	challengeProvider := challenge.ProvideProvider(context, timeProvider, tenantConfiguration)
	anonymousFlow := &flows.AnonymousFlow{
		Interactions: interactionProvider,
		Anonymous:    anonymousProvider,
		Challenges:   challengeProvider,
	}
	idTokenIssuer := oidc.ProvideIDTokenIssuer(tenantConfiguration, urlprefixProvider, authinfoStore, userprofileStore, timeProvider)
	tokenGenerator := _wireTokenGeneratorValue
	tokenHandler := handler2.ProvideTokenHandler(r, tenantConfiguration, factory, authorizationStore, grantStore, grantStore, grantStore, accessEventProvider, sessionProvider, anonymousFlow, idTokenIssuer, tokenGenerator, timeProvider)
	insecureCookieConfig := auth.ProvideSessionInsecureCookieConfig(m)
	cookieConfiguration := session.ProvideSessionCookieConfiguration(r, insecureCookieConfig, tenantConfiguration)
	userController := flows.ProvideUserController(authinfoStore, userprofileStore, tokenHandler, cookieConfiguration, sessionProvider, hookProvider, timeProvider, tenantConfiguration)
	authAPIFlow := &flows.AuthAPIFlow{
		Interactions:   interactionProvider,
		UserController: userController,
	}
	httpHandler := provideLoginHandler(requireAuthz, validator, authAPIFlow, txContext)
	return httpHandler
}

var (
	_wireTokenGeneratorValue = handler2.TokenGenerator(oauth2.GenerateToken)
)

func newSignupHandler(r *http.Request, m auth.DependencyMap) http.Handler {
	context := auth.ProvideContext(r)
	requestID := auth.ProvideLoggingRequestID(r)
	tenantConfiguration := auth.ProvideTenantConfig(context, m)
	factory := logging.ProvideLoggerFactory(context, requestID, tenantConfiguration)
	requireAuthz := handler.NewRequireAuthzFactory(factory)
	validator := auth.ProvideValidator(m)
	timeProvider := time.NewProvider()
	store := redis.ProvideStore(context, tenantConfiguration, timeProvider)
	sqlBuilderFactory := db.ProvideSQLBuilderFactory(tenantConfiguration)
	sqlBuilder := auth.ProvideAuthSQLBuilder(sqlBuilderFactory)
	sqlExecutor := db.ProvideSQLExecutor(context, tenantConfiguration)
	reservedNameChecker := auth.ProvideReservedNameChecker(m)
	typeCheckerFactory := loginid.ProvideTypeCheckerFactory(tenantConfiguration, reservedNameChecker)
	checker := loginid.ProvideChecker(tenantConfiguration, typeCheckerFactory)
	normalizerFactory := loginid.ProvideNormalizerFactory(tenantConfiguration)
	loginidProvider := loginid.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration, checker, normalizerFactory)
	oauthProvider := oauth.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider)
	anonymousProvider := anonymous.ProvideProvider(sqlBuilder, sqlExecutor)
	providerProvider := provider.ProvideProvider(tenantConfiguration, loginidProvider, oauthProvider, anonymousProvider)
	historyStoreImpl := password.ProvideHistoryStore(timeProvider, sqlBuilder, sqlExecutor)
	passwordChecker := password.ProvideChecker(tenantConfiguration, historyStoreImpl)
	passwordProvider := password.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, factory, historyStoreImpl, passwordChecker, tenantConfiguration)
	totpProvider := totp.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	engine := auth.ProvideTemplateEngine(tenantConfiguration, m)
	urlprefixProvider := urlprefix.NewProvider(r)
	txContext := db.ProvideTxContext(context, tenantConfiguration)
	executor := auth.ProvideTaskExecutor(m)
	queue := async.ProvideTaskQueue(context, txContext, requestID, tenantConfiguration, executor)
	oobProvider := oob.ProvideProvider(tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider, engine, urlprefixProvider, queue)
	bearertokenProvider := bearertoken.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	recoverycodeProvider := recoverycode.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	provider3 := &provider2.Provider{
		Password:     passwordProvider,
		TOTP:         totpProvider,
		OOBOTP:       oobProvider,
		BearerToken:  bearertokenProvider,
		RecoveryCode: recoverycodeProvider,
	}
	authinfoStore := pq.ProvideStore(sqlBuilderFactory, sqlExecutor)
	userprofileStore := userprofile.ProvideStore(timeProvider, sqlBuilder, sqlExecutor)
	hookProvider := hook.ProvideHookProvider(context, sqlBuilder, sqlExecutor, requestID, tenantConfiguration, txContext, timeProvider, authinfoStore, userprofileStore, loginidProvider, factory)
	welcomemessageProvider := welcomemessage.ProvideProvider(tenantConfiguration, engine, queue)
	userProvider := interaction.ProvideUserProvider(authinfoStore, userprofileStore, timeProvider, hookProvider, urlprefixProvider, queue, tenantConfiguration, welcomemessageProvider)
	interactionProvider := interaction.ProvideProvider(store, timeProvider, factory, providerProvider, provider3, userProvider, oobProvider, tenantConfiguration, hookProvider)
	authorizationStore := &pq2.AuthorizationStore{
		SQLBuilder:  sqlBuilder,
		SQLExecutor: sqlExecutor,
	}
	grantStore := redis2.ProvideGrantStore(context, factory, tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider)
	eventStore := redis3.ProvideEventStore(context, tenantConfiguration)
	accessEventProvider := auth2.AccessEventProvider{
		Store: eventStore,
	}
	sessionStore := redis4.ProvideStore(context, tenantConfiguration, timeProvider, factory)
	authAccessEventProvider := &auth2.AccessEventProvider{
		Store: eventStore,
	}
	sessionProvider := session.ProvideSessionProvider(r, sessionStore, authAccessEventProvider, tenantConfiguration)
	challengeProvider := challenge.ProvideProvider(context, timeProvider, tenantConfiguration)
	anonymousFlow := &flows.AnonymousFlow{
		Interactions: interactionProvider,
		Anonymous:    anonymousProvider,
		Challenges:   challengeProvider,
	}
	idTokenIssuer := oidc.ProvideIDTokenIssuer(tenantConfiguration, urlprefixProvider, authinfoStore, userprofileStore, timeProvider)
	tokenGenerator := _wireTokenGeneratorValue
	tokenHandler := handler2.ProvideTokenHandler(r, tenantConfiguration, factory, authorizationStore, grantStore, grantStore, grantStore, accessEventProvider, sessionProvider, anonymousFlow, idTokenIssuer, tokenGenerator, timeProvider)
	insecureCookieConfig := auth.ProvideSessionInsecureCookieConfig(m)
	cookieConfiguration := session.ProvideSessionCookieConfiguration(r, insecureCookieConfig, tenantConfiguration)
	userController := flows.ProvideUserController(authinfoStore, userprofileStore, tokenHandler, cookieConfiguration, sessionProvider, hookProvider, timeProvider, tenantConfiguration)
	authAPIFlow := &flows.AuthAPIFlow{
		Interactions:   interactionProvider,
		UserController: userController,
	}
	httpHandler := provideSignupHandler(requireAuthz, validator, authAPIFlow, txContext)
	return httpHandler
}

func newLogoutHandler(r *http.Request, m auth.DependencyMap) http.Handler {
	context := auth.ProvideContext(r)
	requestID := auth.ProvideLoggingRequestID(r)
	tenantConfiguration := auth.ProvideTenantConfig(context, m)
	factory := logging.ProvideLoggerFactory(context, requestID, tenantConfiguration)
	requireAuthz := handler.NewRequireAuthzFactory(factory)
	sqlBuilderFactory := db.ProvideSQLBuilderFactory(tenantConfiguration)
	sqlExecutor := db.ProvideSQLExecutor(context, tenantConfiguration)
	store := pq.ProvideStore(sqlBuilderFactory, sqlExecutor)
	timeProvider := time.NewProvider()
	sqlBuilder := auth.ProvideAuthSQLBuilder(sqlBuilderFactory)
	userprofileStore := userprofile.ProvideStore(timeProvider, sqlBuilder, sqlExecutor)
	txContext := db.ProvideTxContext(context, tenantConfiguration)
	reservedNameChecker := auth.ProvideReservedNameChecker(m)
	typeCheckerFactory := loginid.ProvideTypeCheckerFactory(tenantConfiguration, reservedNameChecker)
	checker := loginid.ProvideChecker(tenantConfiguration, typeCheckerFactory)
	normalizerFactory := loginid.ProvideNormalizerFactory(tenantConfiguration)
	loginidProvider := loginid.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration, checker, normalizerFactory)
	hookProvider := hook.ProvideHookProvider(context, sqlBuilder, sqlExecutor, requestID, tenantConfiguration, txContext, timeProvider, store, userprofileStore, loginidProvider, factory)
	sessionStore := redis4.ProvideStore(context, tenantConfiguration, timeProvider, factory)
	insecureCookieConfig := auth.ProvideSessionInsecureCookieConfig(m)
	cookieConfiguration := session.ProvideSessionCookieConfiguration(r, insecureCookieConfig, tenantConfiguration)
	manager := session.ProvideSessionManager(sessionStore, timeProvider, tenantConfiguration, cookieConfiguration)
	grantStore := redis2.ProvideGrantStore(context, factory, tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider)
	sessionManager := &oauth2.SessionManager{
		Store: grantStore,
		Time:  timeProvider,
	}
	authSessionManager := &auth2.SessionManager{
		AuthInfoStore:       store,
		UserProfileStore:    userprofileStore,
		Hooks:               hookProvider,
		IDPSessions:         manager,
		AccessTokenSessions: sessionManager,
	}
	httpHandler := provideLogoutHandler(requireAuthz, authSessionManager, txContext)
	return httpHandler
}

func newRefreshHandler(r *http.Request, m auth.DependencyMap) http.Handler {
	context := auth.ProvideContext(r)
	requestID := auth.ProvideLoggingRequestID(r)
	tenantConfiguration := auth.ProvideTenantConfig(context, m)
	factory := logging.ProvideLoggerFactory(context, requestID, tenantConfiguration)
	requireAuthz := handler.NewRequireAuthzFactory(factory)
	validator := auth.ProvideValidator(m)
	sqlBuilderFactory := db.ProvideSQLBuilderFactory(tenantConfiguration)
	sqlBuilder := auth.ProvideAuthSQLBuilder(sqlBuilderFactory)
	sqlExecutor := db.ProvideSQLExecutor(context, tenantConfiguration)
	authorizationStore := &pq2.AuthorizationStore{
		SQLBuilder:  sqlBuilder,
		SQLExecutor: sqlExecutor,
	}
	timeProvider := time.NewProvider()
	grantStore := redis2.ProvideGrantStore(context, factory, tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider)
	eventStore := redis3.ProvideEventStore(context, tenantConfiguration)
	accessEventProvider := auth2.AccessEventProvider{
		Store: eventStore,
	}
	store := redis4.ProvideStore(context, tenantConfiguration, timeProvider, factory)
	authAccessEventProvider := &auth2.AccessEventProvider{
		Store: eventStore,
	}
	sessionProvider := session.ProvideSessionProvider(r, store, authAccessEventProvider, tenantConfiguration)
	redisStore := redis.ProvideStore(context, tenantConfiguration, timeProvider)
	reservedNameChecker := auth.ProvideReservedNameChecker(m)
	typeCheckerFactory := loginid.ProvideTypeCheckerFactory(tenantConfiguration, reservedNameChecker)
	checker := loginid.ProvideChecker(tenantConfiguration, typeCheckerFactory)
	normalizerFactory := loginid.ProvideNormalizerFactory(tenantConfiguration)
	loginidProvider := loginid.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration, checker, normalizerFactory)
	oauthProvider := oauth.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider)
	anonymousProvider := anonymous.ProvideProvider(sqlBuilder, sqlExecutor)
	providerProvider := provider.ProvideProvider(tenantConfiguration, loginidProvider, oauthProvider, anonymousProvider)
	historyStoreImpl := password.ProvideHistoryStore(timeProvider, sqlBuilder, sqlExecutor)
	passwordChecker := password.ProvideChecker(tenantConfiguration, historyStoreImpl)
	passwordProvider := password.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, factory, historyStoreImpl, passwordChecker, tenantConfiguration)
	totpProvider := totp.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	engine := auth.ProvideTemplateEngine(tenantConfiguration, m)
	urlprefixProvider := urlprefix.NewProvider(r)
	txContext := db.ProvideTxContext(context, tenantConfiguration)
	executor := auth.ProvideTaskExecutor(m)
	queue := async.ProvideTaskQueue(context, txContext, requestID, tenantConfiguration, executor)
	oobProvider := oob.ProvideProvider(tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider, engine, urlprefixProvider, queue)
	bearertokenProvider := bearertoken.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	recoverycodeProvider := recoverycode.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	provider3 := &provider2.Provider{
		Password:     passwordProvider,
		TOTP:         totpProvider,
		OOBOTP:       oobProvider,
		BearerToken:  bearertokenProvider,
		RecoveryCode: recoverycodeProvider,
	}
	authinfoStore := pq.ProvideStore(sqlBuilderFactory, sqlExecutor)
	userprofileStore := userprofile.ProvideStore(timeProvider, sqlBuilder, sqlExecutor)
	hookProvider := hook.ProvideHookProvider(context, sqlBuilder, sqlExecutor, requestID, tenantConfiguration, txContext, timeProvider, authinfoStore, userprofileStore, loginidProvider, factory)
	welcomemessageProvider := welcomemessage.ProvideProvider(tenantConfiguration, engine, queue)
	userProvider := interaction.ProvideUserProvider(authinfoStore, userprofileStore, timeProvider, hookProvider, urlprefixProvider, queue, tenantConfiguration, welcomemessageProvider)
	interactionProvider := interaction.ProvideProvider(redisStore, timeProvider, factory, providerProvider, provider3, userProvider, oobProvider, tenantConfiguration, hookProvider)
	challengeProvider := challenge.ProvideProvider(context, timeProvider, tenantConfiguration)
	anonymousFlow := &flows.AnonymousFlow{
		Interactions: interactionProvider,
		Anonymous:    anonymousProvider,
		Challenges:   challengeProvider,
	}
	idTokenIssuer := oidc.ProvideIDTokenIssuer(tenantConfiguration, urlprefixProvider, authinfoStore, userprofileStore, timeProvider)
	tokenGenerator := _wireTokenGeneratorValue
	tokenHandler := handler2.ProvideTokenHandler(r, tenantConfiguration, factory, authorizationStore, grantStore, grantStore, grantStore, accessEventProvider, sessionProvider, anonymousFlow, idTokenIssuer, tokenGenerator, timeProvider)
	httpHandler := provideRefreshHandler(requireAuthz, validator, tokenHandler, txContext)
	return httpHandler
}

func newChangePasswordHandler(r *http.Request, m auth.DependencyMap) http.Handler {
	context := auth.ProvideContext(r)
	requestID := auth.ProvideLoggingRequestID(r)
	tenantConfiguration := auth.ProvideTenantConfig(context, m)
	factory := logging.ProvideLoggerFactory(context, requestID, tenantConfiguration)
	requireAuthz := handler.NewRequireAuthzFactory(factory)
	validator := auth.ProvideValidator(m)
	sqlBuilderFactory := db.ProvideSQLBuilderFactory(tenantConfiguration)
	sqlExecutor := db.ProvideSQLExecutor(context, tenantConfiguration)
	store := pq.ProvideStore(sqlBuilderFactory, sqlExecutor)
	txContext := db.ProvideTxContext(context, tenantConfiguration)
	timeProvider := time.NewProvider()
	sqlBuilder := auth.ProvideAuthSQLBuilder(sqlBuilderFactory)
	userprofileStore := userprofile.ProvideStore(timeProvider, sqlBuilder, sqlExecutor)
	reservedNameChecker := auth.ProvideReservedNameChecker(m)
	typeCheckerFactory := loginid.ProvideTypeCheckerFactory(tenantConfiguration, reservedNameChecker)
	checker := loginid.ProvideChecker(tenantConfiguration, typeCheckerFactory)
	normalizerFactory := loginid.ProvideNormalizerFactory(tenantConfiguration)
	loginidProvider := loginid.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration, checker, normalizerFactory)
	hookProvider := hook.ProvideHookProvider(context, sqlBuilder, sqlExecutor, requestID, tenantConfiguration, txContext, timeProvider, store, userprofileStore, loginidProvider, factory)
	executor := auth.ProvideTaskExecutor(m)
	queue := async.ProvideTaskQueue(context, txContext, requestID, tenantConfiguration, executor)
	redisStore := redis.ProvideStore(context, tenantConfiguration, timeProvider)
	oauthProvider := oauth.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider)
	anonymousProvider := anonymous.ProvideProvider(sqlBuilder, sqlExecutor)
	providerProvider := provider.ProvideProvider(tenantConfiguration, loginidProvider, oauthProvider, anonymousProvider)
	historyStoreImpl := password.ProvideHistoryStore(timeProvider, sqlBuilder, sqlExecutor)
	passwordChecker := password.ProvideChecker(tenantConfiguration, historyStoreImpl)
	passwordProvider := password.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, factory, historyStoreImpl, passwordChecker, tenantConfiguration)
	totpProvider := totp.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	engine := auth.ProvideTemplateEngine(tenantConfiguration, m)
	urlprefixProvider := urlprefix.NewProvider(r)
	oobProvider := oob.ProvideProvider(tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider, engine, urlprefixProvider, queue)
	bearertokenProvider := bearertoken.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	recoverycodeProvider := recoverycode.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	provider3 := &provider2.Provider{
		Password:     passwordProvider,
		TOTP:         totpProvider,
		OOBOTP:       oobProvider,
		BearerToken:  bearertokenProvider,
		RecoveryCode: recoverycodeProvider,
	}
	welcomemessageProvider := welcomemessage.ProvideProvider(tenantConfiguration, engine, queue)
	userProvider := interaction.ProvideUserProvider(store, userprofileStore, timeProvider, hookProvider, urlprefixProvider, queue, tenantConfiguration, welcomemessageProvider)
	interactionProvider := interaction.ProvideProvider(redisStore, timeProvider, factory, providerProvider, provider3, userProvider, oobProvider, tenantConfiguration, hookProvider)
	passwordFlow := &flows.PasswordFlow{
		Interactions: interactionProvider,
	}
	httpHandler := provideChangePasswordHandler(requireAuthz, validator, store, txContext, userprofileStore, hookProvider, queue, passwordFlow)
	return httpHandler
}

func newResetPasswordHandler(r *http.Request, m auth.DependencyMap) http.Handler {
	context := auth.ProvideContext(r)
	requestID := auth.ProvideLoggingRequestID(r)
	tenantConfiguration := auth.ProvideTenantConfig(context, m)
	factory := logging.ProvideLoggerFactory(context, requestID, tenantConfiguration)
	requireAuthz := handler.NewRequireAuthzFactory(factory)
	validator := auth.ProvideValidator(m)
	timeProvider := time.NewProvider()
	sqlBuilderFactory := db.ProvideSQLBuilderFactory(tenantConfiguration)
	sqlBuilder := auth.ProvideAuthSQLBuilder(sqlBuilderFactory)
	sqlExecutor := db.ProvideSQLExecutor(context, tenantConfiguration)
	store := userprofile.ProvideStore(timeProvider, sqlBuilder, sqlExecutor)
	authinfoStore := pq.ProvideStore(sqlBuilderFactory, sqlExecutor)
	txContext := db.ProvideTxContext(context, tenantConfiguration)
	executor := auth.ProvideTaskExecutor(m)
	queue := async.ProvideTaskQueue(context, txContext, requestID, tenantConfiguration, executor)
	reservedNameChecker := auth.ProvideReservedNameChecker(m)
	typeCheckerFactory := loginid.ProvideTypeCheckerFactory(tenantConfiguration, reservedNameChecker)
	checker := loginid.ProvideChecker(tenantConfiguration, typeCheckerFactory)
	normalizerFactory := loginid.ProvideNormalizerFactory(tenantConfiguration)
	loginidProvider := loginid.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration, checker, normalizerFactory)
	hookProvider := hook.ProvideHookProvider(context, sqlBuilder, sqlExecutor, requestID, tenantConfiguration, txContext, timeProvider, authinfoStore, store, loginidProvider, factory)
	redisStore := redis.ProvideStore(context, tenantConfiguration, timeProvider)
	oauthProvider := oauth.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider)
	anonymousProvider := anonymous.ProvideProvider(sqlBuilder, sqlExecutor)
	providerProvider := provider.ProvideProvider(tenantConfiguration, loginidProvider, oauthProvider, anonymousProvider)
	historyStoreImpl := password.ProvideHistoryStore(timeProvider, sqlBuilder, sqlExecutor)
	passwordChecker := password.ProvideChecker(tenantConfiguration, historyStoreImpl)
	passwordProvider := password.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, factory, historyStoreImpl, passwordChecker, tenantConfiguration)
	totpProvider := totp.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	engine := auth.ProvideTemplateEngine(tenantConfiguration, m)
	urlprefixProvider := urlprefix.NewProvider(r)
	oobProvider := oob.ProvideProvider(tenantConfiguration, sqlBuilder, sqlExecutor, timeProvider, engine, urlprefixProvider, queue)
	bearertokenProvider := bearertoken.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	recoverycodeProvider := recoverycode.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration)
	provider3 := &provider2.Provider{
		Password:     passwordProvider,
		TOTP:         totpProvider,
		OOBOTP:       oobProvider,
		BearerToken:  bearertokenProvider,
		RecoveryCode: recoverycodeProvider,
	}
	welcomemessageProvider := welcomemessage.ProvideProvider(tenantConfiguration, engine, queue)
	userProvider := interaction.ProvideUserProvider(authinfoStore, store, timeProvider, hookProvider, urlprefixProvider, queue, tenantConfiguration, welcomemessageProvider)
	interactionProvider := interaction.ProvideProvider(redisStore, timeProvider, factory, providerProvider, provider3, userProvider, oobProvider, tenantConfiguration, hookProvider)
	passwordFlow := &flows.PasswordFlow{
		Interactions: interactionProvider,
	}
	httpHandler := provideResetPasswordHandler(requireAuthz, validator, store, authinfoStore, txContext, queue, hookProvider, passwordFlow)
	return httpHandler
}

func newListIdentitiesHandler(r *http.Request, m auth.DependencyMap) http.Handler {
	context := auth.ProvideContext(r)
	requestID := auth.ProvideLoggingRequestID(r)
	tenantConfiguration := auth.ProvideTenantConfig(context, m)
	factory := logging.ProvideLoggerFactory(context, requestID, tenantConfiguration)
	requireAuthz := handler.NewRequireAuthzFactory(factory)
	txContext := db.ProvideTxContext(context, tenantConfiguration)
	sqlBuilderFactory := db.ProvideSQLBuilderFactory(tenantConfiguration)
	sqlBuilder := auth.ProvideAuthSQLBuilder(sqlBuilderFactory)
	sqlExecutor := db.ProvideSQLExecutor(context, tenantConfiguration)
	timeProvider := time.NewProvider()
	reservedNameChecker := auth.ProvideReservedNameChecker(m)
	typeCheckerFactory := loginid.ProvideTypeCheckerFactory(tenantConfiguration, reservedNameChecker)
	checker := loginid.ProvideChecker(tenantConfiguration, typeCheckerFactory)
	normalizerFactory := loginid.ProvideNormalizerFactory(tenantConfiguration)
	loginidProvider := loginid.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider, tenantConfiguration, checker, normalizerFactory)
	oauthProvider := oauth.ProvideProvider(sqlBuilder, sqlExecutor, timeProvider)
	anonymousProvider := anonymous.ProvideProvider(sqlBuilder, sqlExecutor)
	providerProvider := provider.ProvideProvider(tenantConfiguration, loginidProvider, oauthProvider, anonymousProvider)
	httpHandler := providerListIdentitiesHandler(requireAuthz, txContext, providerProvider)
	return httpHandler
}

// wire.go:

func provideLoginHandler(
	requireAuthz handler.RequireAuthz,
	v *validation.Validator,
	f LoginInteractionFlow,
	tx db.TxContext,
) http.Handler {
	h := &LoginHandler{
		Validator:    v,
		Interactions: f,
		TxContext:    tx,
	}
	return requireAuthz(h, h)
}

func provideSignupHandler(
	requireAuthz handler.RequireAuthz,
	v *validation.Validator,
	f SignupInteractionFlow,
	tx db.TxContext,
) http.Handler {
	h := &SignupHandler{
		Validator:    v,
		Interactions: f,
		TxContext:    tx,
	}
	return requireAuthz(h, h)
}

func provideLogoutHandler(
	requireAuthz handler.RequireAuthz,
	sm logoutSessionManager,
	tx db.TxContext,
) http.Handler {
	h := &LogoutHandler{
		SessionManager: sm,
		TxContext:      tx,
	}
	return requireAuthz(h, h)
}

func provideRefreshHandler(
	requireAuthz handler.RequireAuthz,
	v *validation.Validator,
	rp refreshProvider,
	tx db.TxContext,
) http.Handler {
	h := &RefreshHandler{
		validator:       v,
		refreshProvider: rp,
		txContext:       tx,
	}
	return requireAuthz(h, h)
}

func provideChangePasswordHandler(
	requireAuthz handler.RequireAuthz,
	v *validation.Validator,
	as authinfo.Store,
	tx db.TxContext,
	ups userprofile.Store,
	hp hook.Provider,
	aq async.Queue,
	f PasswordFlow,
) http.Handler {
	h := &ChangePasswordHandler{
		Validator:        v,
		AuthInfoStore:    as,
		TxContext:        tx,
		UserProfileStore: ups,
		HookProvider:     hp,
		TaskQueue:        aq,
		Interactions:     f,
	}
	return requireAuthz(h, h)
}

func provideResetPasswordHandler(
	requireAuthz handler.RequireAuthz,
	v *validation.Validator,
	ups userprofile.Store,
	as authinfo.Store,
	tx db.TxContext,
	aq async.Queue,
	hp hook.Provider,
	f ResetPasswordFlow,
) http.Handler {
	h := &ResetPasswordHandler{
		Validator:        v,
		UserProfileStore: ups,
		AuthInfoStore:    as,
		TxContext:        tx,
		TaskQueue:        aq,
		HookProvider:     hp,
		Interactions:     f,
	}
	return requireAuthz(h, h)
}

func providerListIdentitiesHandler(
	requireAuthz handler.RequireAuthz,
	tx db.TxContext,
	ip ListIdentityProvider,
) http.Handler {
	h := &ListIdentitiesHandler{
		TxContext:        tx,
		IdentityProvider: ip,
	}
	return requireAuthz(h, h)
}
