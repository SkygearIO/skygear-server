package session

import (
	"net/http"
	"time"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/config"
	coreHttp "github.com/skygeario/skygear-server/pkg/core/http"
	"github.com/skygeario/skygear-server/pkg/core/model"
)

type writerImpl struct {
	authContext       auth.ContextGetter
	clientConfigs     []config.APIClientConfiguration
	mfaConfiguration  *config.MFAConfiguration
	useInsecureCookie bool
}

func NewWriter(
	authContext auth.ContextGetter,
	clientConfigs []config.APIClientConfiguration,
	mfaConfiguration *config.MFAConfiguration,
	useInsecureCookie bool,
) Writer {
	return &writerImpl{
		authContext:       authContext,
		clientConfigs:     clientConfigs,
		mfaConfiguration:  mfaConfiguration,
		useInsecureCookie: useInsecureCookie,
	}
}

func (w *writerImpl) WriteSession(rw http.ResponseWriter, accessToken *string, mfaBearerToken *string) {
	clientConfig, _ := model.GetClientConfig(w.clientConfigs, w.authContext.AccessKey().ClientID)
	useCookie := clientConfig.SessionTransport == config.SessionTransportTypeCookie

	cookieSession := &http.Cookie{
		Name:     coreHttp.CookieNameSession,
		Path:     "/",
		HttpOnly: true,
		Secure:   !w.useInsecureCookie,
	}
	cookieMFABearerToken := &http.Cookie{
		Name:     coreHttp.CookieNameMFABearerToken,
		Path:     "/_auth/mfa/bearer_token/authenticate",
		HttpOnly: true,
		Secure:   !w.useInsecureCookie,
	}
	w.configureCookieSameSite(cookieSession, clientConfig.SameSite)
	w.configureCookieSameSite(cookieMFABearerToken, clientConfig.SameSite)

	if useCookie {
		cookieSession.Value = *accessToken
		*accessToken = ""
		cookieSession.MaxAge = clientConfig.AccessTokenLifetime

		if mfaBearerToken != nil {
			cookieMFABearerToken.Value = *mfaBearerToken
			*mfaBearerToken = ""
			cookieMFABearerToken.MaxAge = 86400 * w.mfaConfiguration.BearerToken.ExpireInDays
		}
	} else {
		cookieSession.Expires = time.Unix(0, 0)
		cookieMFABearerToken.Expires = time.Unix(0, 0)
	}

	coreHttp.UpdateCookie(rw, cookieSession)
	if mfaBearerToken != nil {
		coreHttp.UpdateCookie(rw, cookieMFABearerToken)
	}
}

func (w *writerImpl) configureCookieSameSite(cookie *http.Cookie, sameSite config.SessionCookieSameSite) {
	switch sameSite {
	case config.SessionCookieSameSiteNone:
		cookie.SameSite = http.SameSiteDefaultMode
	case config.SessionCookieSameSiteLax:
		cookie.SameSite = http.SameSiteLaxMode
	case config.SessionCookieSameSiteStrict:
		cookie.SameSite = http.SameSiteStrictMode
	}
}

func (w *writerImpl) ClearSession(rw http.ResponseWriter) {
	coreHttp.UpdateCookie(rw, &http.Cookie{
		Name:     coreHttp.CookieNameSession,
		Path:     "/",
		HttpOnly: true,
		Secure:   !w.useInsecureCookie,
		Expires:  time.Unix(0, 0),
	})
}

func (w *writerImpl) ClearMFABearerToken(rw http.ResponseWriter) {
	coreHttp.UpdateCookie(rw, &http.Cookie{
		Name:     coreHttp.CookieNameMFABearerToken,
		Path:     "/_auth/mfa/bearer_token/authenticate",
		HttpOnly: true,
		Secure:   !w.useInsecureCookie,
		Expires:  time.Unix(0, 0),
	})
}
