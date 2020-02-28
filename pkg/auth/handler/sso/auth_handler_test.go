package sso

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/authnsession"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/mfa"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/oauth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/password"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/sso"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/apiclientconfig"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/metadata"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	authtest "github.com/skygeario/skygear-server/pkg/core/auth/testing"
	coreconfig "github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/crypto"
	"github.com/skygeario/skygear-server/pkg/core/db"
	coreHttp "github.com/skygeario/skygear-server/pkg/core/http"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
	coreTime "github.com/skygeario/skygear-server/pkg/core/time"

	. "github.com/skygeario/skygear-server/pkg/core/skytest"
	. "github.com/smartystreets/goconvey/convey"
)

func decodeResultInURL(urlString string) ([]byte, error) {
	u, err := url.Parse(urlString)
	if err != nil {
		return nil, err
	}
	result := u.Query().Get("x-skygear-result")
	bytes, err := base64.StdEncoding.DecodeString(result)
	if err != nil {
		return nil, err
	}
	var j map[string]interface{}
	err = json.Unmarshal(bytes, &j)
	if err != nil {
		return nil, err
	}
	innerResult := j["result"].(map[string]interface{})
	actualResult, ok := innerResult["result"]
	if !ok {
		return bytes, nil
	}
	code, err := sso.DecodeSkygearAuthorizationCode("secret", "myapp", actualResult.(string))
	if err != nil {
		return nil, err
	}
	innerResult["result"] = code
	return json.Marshal(j)
}

func decodeUXModeManualResult(bytes []byte) ([]byte, error) {
	var j map[string]interface{}
	err := json.Unmarshal(bytes, &j)
	if err != nil {
		return nil, err
	}
	code := j["result"].(string)
	authCode, err := sso.DecodeSkygearAuthorizationCode("secret", "myapp", code)
	if err != nil {
		return nil, err
	}
	j["result"] = authCode
	return json.Marshal(j)
}

func TestAuthPayload(t *testing.T) {
	Convey("Test AuthRequestPayload", t, func() {
		Convey("validate valid payload", func() {
			payload := AuthRequestPayload{
				Code:  "code",
				State: "state",
				Nonce: "nonce",
			}
			So(payload.Validate(), ShouldBeNil)
		})

		Convey("validate payload without code", func() {
			payload := AuthRequestPayload{
				State: "state",
			}
			So(payload.Validate(), ShouldBeError)
		})

		Convey("validate payload without state", func() {
			payload := AuthRequestPayload{
				Code: "code",
			}
			So(payload.Validate(), ShouldBeError)
		})
	})
}

func TestAuthHandler(t *testing.T) {
	realTime := timeNow
	timeNow = func() time.Time { return time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC) }
	defer func() {
		timeNow = realTime
	}()

	Convey("Test AuthHandler with login action", t, func() {
		action := "login"
		stateJWTSecret := "secret"
		providerName := "mock"
		providerUserID := "mock_user_id"
		sh := &AuthHandler{}
		sh.TxContext = db.NewMockTxContext()
		sh.APIClientConfigurationProvider = apiclientconfig.NewMockProvider("api_key")
		authContext := authtest.NewMockContext().
			UseUser("faseng.cat.id", "faseng.cat.principal.id").
			MarkVerified()
		sh.AuthContext = authContext
		sh.AuthContextSetter = authContext
		oauthConfig := &coreconfig.OAuthConfiguration{
			StateJWTSecret: stateJWTSecret,
			AllowedCallbackURLs: []string{
				"http://localhost:3000",
			},
		}
		providerConfig := coreconfig.OAuthProviderConfiguration{
			ID:           providerName,
			Type:         "google",
			ClientID:     "mock_client_id",
			ClientSecret: "mock_client_secret",
		}
		mockProvider := sso.MockSSOProvider{
			URLPrefix:      &url.URL{Scheme: "https", Host: "api.example.com"},
			BaseURL:        "http://mock/auth",
			OAuthConfig:    oauthConfig,
			ProviderConfig: providerConfig,
			UserInfo: sso.ProviderUserInfo{
				ID:    providerUserID,
				Email: "mock@example.com",
			},
		}
		sh.OAuthProvider = &mockProvider
		sh.SSOProvider = &mockProvider
		mockOAuthProvider := oauth.NewMockProvider(nil)
		sh.OAuthAuthProvider = mockOAuthProvider
		authInfoStore := authinfo.NewMockStoreWithAuthInfoMap(
			map[string]authinfo.AuthInfo{},
		)
		sh.AuthInfoStore = authInfoStore
		sessionProvider := session.NewMockProvider()
		sessionWriter := session.NewMockWriter()
		userProfileStore := userprofile.NewMockUserProfileStore()
		sh.UserProfileStore = userProfileStore
		sh.AuthHandlerHTMLProvider = sso.NewAuthHandlerHTMLProvider(
			&url.URL{Scheme: "https", Host: "api.example.com"},
		)
		one := 1
		loginIDsKeys := []coreconfig.LoginIDKeyConfiguration{
			coreconfig.LoginIDKeyConfiguration{Key: "email", Maximum: &one},
		}
		allowedRealms := []string{loginid.DefaultRealm}
		passwordAuthProvider := password.NewMockProviderWithPrincipalMap(
			loginIDsKeys,
			allowedRealms,
			map[string]password.Principal{},
		)
		identityProvider := principal.NewMockIdentityProvider(sh.OAuthAuthProvider, passwordAuthProvider)
		sh.IdentityProvider = identityProvider
		hookProvider := hook.NewMockProvider()
		sh.HookProvider = hookProvider
		timeProvider := &coreTime.MockProvider{TimeNowUTC: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)}
		mfaStore := mfa.NewMockStore(timeProvider)
		mfaConfiguration := &coreconfig.MFAConfiguration{
			Enabled:     false,
			Enforcement: coreconfig.MFAEnforcementOptional,
		}
		mfaSender := mfa.NewMockSender()
		mfaProvider := mfa.NewProvider(mfaStore, mfaConfiguration, timeProvider, mfaSender)

		sh.AuthnSessionProvider = authnsession.NewMockProvider(
			mfaConfiguration,
			timeProvider,
			mfaProvider,
			authInfoStore,
			sessionProvider,
			sessionWriter,
			identityProvider,
			hookProvider,
			userProfileStore,
		)

		nonce := "nonce"
		hashedNonce := crypto.SHA256String(nonce)
		nonceCookie := &http.Cookie{
			Name:  coreHttp.CookieNameOpenIDConnectNonce,
			Value: nonce,
		}

		Convey("should write code in the response body if ux_mode is manual", func() {
			uxMode := sso.UXModeManual

			// oauth state
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      uxMode,
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)
			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}

			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()
			sh.ServeHTTP(resp, req)

			p, err := sh.OAuthAuthProvider.GetPrincipalByProvider(oauth.GetByProviderOptions{
				ProviderType:   "google",
				ProviderUserID: providerUserID,
			})
			So(err, ShouldBeNil)

			actual, err := decodeUXModeManualResult(resp.Body.Bytes())
			So(err, ShouldBeNil)
			So(actual, ShouldEqualJSON, fmt.Sprintf(`
			{
				"result": {
					"action": "login",
					"code_challenge": "",
					"user_id": "%s",
					"principal_id": "%s",
					"session_create_reason": "signup"
				}
			}`, p.UserID, p.ID))
		})

		Convey("should return callback url when ux_mode is web_redirect", func() {
			uxMode := sso.UXModeWebRedirect

			// oauth state
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      uxMode,
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)

			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}

			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()

			sh.ServeHTTP(resp, req)
			// for web_redirect, it should redirect to original callback url
			So(resp.Code, ShouldEqual, 302)
			location := resp.Result().Header.Get("Location")
			actual, err := decodeResultInURL(location)
			So(err, ShouldBeNil)

			p, err := sh.OAuthAuthProvider.GetPrincipalByProvider(oauth.GetByProviderOptions{
				ProviderType:   "google",
				ProviderUserID: providerUserID,
			})
			So(err, ShouldBeNil)
			So(actual, ShouldEqualJSON, fmt.Sprintf(`
			{
				"callback_url": "http://localhost:3000",
				"result": {
					"result": {
						"action": "login",
						"code_challenge": "",
						"user_id": "%s",
						"principal_id": "%s",
						"session_create_reason": "signup"
					}
				}
			}`, p.UserID, p.ID))
		})

		Convey("should return html page when ux_mode is web_popup", func() {
			uxMode := sso.UXModeWebPopup

			// oauth state
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      uxMode,
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)

			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}

			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()

			sh.ServeHTTP(resp, req)
			// for web_redirect, it should redirect to original callback url
			So(resp.Code, ShouldEqual, 200)
			apiEndpointPattern := `"https:\\/\\/api.example.com/_auth/sso/config"`
			matched, err := regexp.MatchString(apiEndpointPattern, resp.Body.String())
			So(err, ShouldBeNil)
			So(matched, ShouldBeTrue)
		})

		Convey("should return callback url with result query parameter when ux_mode is mobile_app", func() {
			uxMode := sso.UXModeMobileApp

			// oauth state
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      uxMode,
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)

			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}

			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()

			sh.ServeHTTP(resp, req)
			// for mobile app, it should redirect to original callback url
			So(resp.Code, ShouldEqual, 302)
			// check location result query parameter
			actual, err := decodeResultInURL(resp.Header().Get("Location"))
			So(err, ShouldBeNil)
			p, _ := sh.OAuthAuthProvider.GetPrincipalByProvider(oauth.GetByProviderOptions{
				ProviderType:   "google",
				ProviderUserID: providerUserID,
			})
			So(actual, ShouldEqualJSON, fmt.Sprintf(`{
				"callback_url": "http://localhost:3000",
				"result": {
					"result": {
						"action": "login",
						"code_challenge": "",
						"user_id": "%s",
						"principal_id": "%s",
						"session_create_reason": "signup"
					}
				}
			}`, p.UserID, p.ID))
		})
	})

	Convey("Test AuthHandler with link action", t, func() {
		action := "link"
		stateJWTSecret := "secret"
		providerUserID := "mock_user_id"
		sh := &AuthHandler{}
		sh.APIClientConfigurationProvider = apiclientconfig.NewMockProvider("api_key")
		sh.TxContext = db.NewMockTxContext()
		authContext := authtest.NewMockContext().
			UseUser("faseng.cat.id", "faseng.cat.principal.id").
			MarkVerified()
		sh.AuthContext = authContext
		sh.AuthContextSetter = authContext
		oauthConfig := &coreconfig.OAuthConfiguration{
			StateJWTSecret: stateJWTSecret,
			AllowedCallbackURLs: []string{
				"http://localhost:3000",
			},
		}
		providerConfig := coreconfig.OAuthProviderConfiguration{
			ID:           "mock",
			Type:         "google",
			ClientID:     "mock_client_id",
			ClientSecret: "mock_client_secret",
		}
		mockProvider := sso.MockSSOProvider{
			URLPrefix:      &url.URL{Scheme: "https", Host: "api.example.com"},
			BaseURL:        "http://mock/auth",
			OAuthConfig:    oauthConfig,
			ProviderConfig: providerConfig,
			UserInfo: sso.ProviderUserInfo{
				ID:    providerUserID,
				Email: "mock@example.com",
			},
		}
		sh.OAuthProvider = &mockProvider
		sh.SSOProvider = &mockProvider
		mockOAuthProvider := oauth.NewMockProvider([]*oauth.Principal{
			&oauth.Principal{
				ID:           "jane.doe.id",
				UserID:       "jane.doe.id",
				ProviderType: "google",
				ProviderKeys: map[string]interface{}{},
			},
		})
		sh.OAuthAuthProvider = mockOAuthProvider
		authInfoStore := authinfo.NewMockStoreWithAuthInfoMap(
			map[string]authinfo.AuthInfo{
				"john.doe.id": authinfo.AuthInfo{
					ID: "john.doe.id",
				},
				"jane.doe.id": authinfo.AuthInfo{
					ID: "jane.doe.id",
				},
			},
		)
		sh.AuthInfoStore = authInfoStore
		sessionProvider := session.NewMockProvider()
		sessionWriter := session.NewMockWriter()
		userProfileStore := userprofile.NewMockUserProfileStore()
		sh.UserProfileStore = userProfileStore
		sh.AuthHandlerHTMLProvider = sso.NewAuthHandlerHTMLProvider(
			&url.URL{Scheme: "https", Host: "api.example.com"},
		)
		one := 1
		loginIDsKeys := []coreconfig.LoginIDKeyConfiguration{
			coreconfig.LoginIDKeyConfiguration{Type: "email", Maximum: &one},
		}
		allowedRealms := []string{loginid.DefaultRealm}
		passwordAuthProvider := password.NewMockProviderWithPrincipalMap(
			loginIDsKeys,
			allowedRealms,
			map[string]password.Principal{},
		)
		identityProvider := principal.NewMockIdentityProvider(sh.OAuthAuthProvider, passwordAuthProvider)
		sh.IdentityProvider = identityProvider
		hookProvider := hook.NewMockProvider()
		sh.HookProvider = hookProvider
		timeProvider := &coreTime.MockProvider{TimeNowUTC: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)}
		mfaStore := mfa.NewMockStore(timeProvider)
		mfaConfiguration := &coreconfig.MFAConfiguration{
			Enabled:     false,
			Enforcement: coreconfig.MFAEnforcementOptional,
		}
		mfaSender := mfa.NewMockSender()
		mfaProvider := mfa.NewProvider(mfaStore, mfaConfiguration, timeProvider, mfaSender)
		sh.AuthnSessionProvider = authnsession.NewMockProvider(
			mfaConfiguration,
			timeProvider,
			mfaProvider,
			authInfoStore,
			sessionProvider,
			sessionWriter,
			identityProvider,
			hookProvider,
			userProfileStore,
		)

		nonce := "nonce"
		hashedNonce := crypto.SHA256String(nonce)
		nonceCookie := &http.Cookie{
			Name:  coreHttp.CookieNameOpenIDConnectNonce,
			Value: nonce,
		}

		Convey("should return callback url when ux_mode is web_redirect", func() {
			mockOAuthProvider := oauth.NewMockProvider(nil)
			sh.OAuthAuthProvider = mockOAuthProvider
			uxMode := sso.UXModeWebRedirect

			// oauth state
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      uxMode,
				},
				LinkState: sso.LinkState{
					UserID: "john.doe.id",
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)

			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}

			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()

			sh.ServeHTTP(resp, req)
			// for web_redirect, it should redirect to original callback url
			So(resp.Code, ShouldEqual, 302)

			actual, err := decodeResultInURL(resp.Header().Get("Location"))
			So(err, ShouldBeNil)
			p, _ := sh.OAuthAuthProvider.GetPrincipalByProvider(oauth.GetByProviderOptions{
				ProviderType:   "google",
				ProviderUserID: providerUserID,
			})
			So(actual, ShouldEqualJSON, fmt.Sprintf(`
			{
				"callback_url": "http://localhost:3000",
				"result": {
					"result": {
						"action": "link",
						"code_challenge": "",
						"user_id": "john.doe.id",
						"principal_id": "%s"
					}
				}
			}
			`, p.ID))
		})

		Convey("should get err if user is already linked", func() {
			uxMode := sso.UXModeWebRedirect
			mockOAuthProvider := oauth.NewMockProvider([]*oauth.Principal{
				&oauth.Principal{
					ID:             "jane.doe.id",
					UserID:         "jane.doe.id",
					ProviderType:   "google",
					ProviderKeys:   map[string]interface{}{},
					ProviderUserID: providerUserID,
				},
			})
			sh.OAuthAuthProvider = mockOAuthProvider

			// oauth state
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      uxMode,
				},
				LinkState: sso.LinkState{
					UserID: "jane.doe.id",
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)

			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}

			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()

			sh.ServeHTTP(resp, req)
			So(resp.Code, ShouldEqual, 302)

			actual, err := decodeResultInURL(resp.Header().Get("Location"))
			So(err, ShouldBeNil)
			So(actual, ShouldEqualJSON, `
			{
				"callback_url": "http://localhost:3000",
				"result": {
					"error": {
						"name": "Unauthorized",
						"reason": "SSOFailed",
						"message": "user is already linked to this provider",
						"code": 401,
						"info": { "cause": { "kind": "AlreadyLinked" } }
					}
				}
			}
			`)
		})
	})

	Convey("Test OnUserDuplicate", t, func() {
		action := "login"
		UXMode := sso.UXModeWebRedirect
		stateJWTSecret := "secret"
		providerName := "mock"
		providerUserID := "mock_user_id"

		sh := &AuthHandler{}
		sh.APIClientConfigurationProvider = apiclientconfig.NewMockProvider("api_key")
		sh.TxContext = db.NewMockTxContext()
		authContext := authtest.NewMockContext().
			UseUser("faseng.cat.id", "faseng.cat.principal.id").
			MarkVerified()
		sh.AuthContext = authContext
		sh.AuthContextSetter = authContext
		oauthConfig := &coreconfig.OAuthConfiguration{
			StateJWTSecret: stateJWTSecret,
			AllowedCallbackURLs: []string{
				"http://localhost:3000",
			},
		}
		providerConfig := coreconfig.OAuthProviderConfiguration{
			ID:           providerName,
			Type:         "google",
			ClientID:     "mock_client_id",
			ClientSecret: "mock_client_secret",
		}
		mockProvider := sso.MockSSOProvider{
			URLPrefix:      &url.URL{Scheme: "https", Host: "api.example.com"},
			BaseURL:        "http://mock/auth",
			OAuthConfig:    oauthConfig,
			ProviderConfig: providerConfig,
			UserInfo: sso.ProviderUserInfo{ID: providerUserID,
				Email: "john.doe@example.com"},
		}
		sh.OAuthProvider = &mockProvider
		sh.SSOProvider = &mockProvider
		mockOAuthProvider := oauth.NewMockProvider(nil)
		sh.OAuthAuthProvider = mockOAuthProvider
		authInfoStore := authinfo.NewMockStoreWithAuthInfoMap(
			map[string]authinfo.AuthInfo{
				"john.doe.id": authinfo.AuthInfo{
					ID:         "john.doe.id",
					VerifyInfo: map[string]bool{},
				},
			},
		)
		sh.AuthInfoStore = authInfoStore
		sessionProvider := session.NewMockProvider()
		sessionWriter := session.NewMockWriter()
		profileData := map[string]map[string]interface{}{
			"john.doe.id": map[string]interface{}{},
		}
		userProfileStore := userprofile.NewMockUserProfileStoreByData(profileData)
		sh.UserProfileStore = userProfileStore
		sh.AuthHandlerHTMLProvider = sso.NewAuthHandlerHTMLProvider(
			&url.URL{Scheme: "https", Host: "api.example.com"},
		)
		one := 1
		loginIDsKeys := []coreconfig.LoginIDKeyConfiguration{
			coreconfig.LoginIDKeyConfiguration{
				Key:     "email",
				Type:    coreconfig.LoginIDKeyType(metadata.Email),
				Maximum: &one,
			},
		}
		allowedRealms := []string{loginid.DefaultRealm}
		passwordAuthProvider := password.NewMockProviderWithPrincipalMap(
			loginIDsKeys,
			allowedRealms,
			map[string]password.Principal{
				"john.doe.principal.id": password.Principal{
					ID:             "john.doe.principal.id",
					UserID:         "john.doe.id",
					LoginIDKey:     "email",
					LoginID:        "john.doe@example.com",
					Realm:          "default",
					HashedPassword: []byte("$2a$10$/jm/S1sY6ldfL6UZljlJdOAdJojsJfkjg/pqK47Q8WmOLE19tGWQi"), // 123456
					ClaimsValue: map[string]interface{}{
						"email": "john.doe@example.com",
					},
				},
			},
		)
		identityProvider := principal.NewMockIdentityProvider(sh.OAuthAuthProvider, passwordAuthProvider)
		sh.IdentityProvider = identityProvider
		hookProvider := hook.NewMockProvider()
		sh.HookProvider = hookProvider
		timeProvider := &coreTime.MockProvider{TimeNowUTC: time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)}
		mfaStore := mfa.NewMockStore(timeProvider)
		mfaConfiguration := &coreconfig.MFAConfiguration{
			Enabled:     false,
			Enforcement: coreconfig.MFAEnforcementOptional,
		}
		mfaSender := mfa.NewMockSender()
		mfaProvider := mfa.NewProvider(mfaStore, mfaConfiguration, timeProvider, mfaSender)
		sh.AuthnSessionProvider = authnsession.NewMockProvider(
			mfaConfiguration,
			timeProvider,
			mfaProvider,
			authInfoStore,
			sessionProvider,
			sessionWriter,
			identityProvider,
			hookProvider,
			userProfileStore,
		)

		nonce := "nonce"
		hashedNonce := crypto.SHA256String(nonce)
		nonceCookie := &http.Cookie{
			Name:  coreHttp.CookieNameOpenIDConnectNonce,
			Value: nonce,
		}

		Convey("OnUserDuplicate == abort", func() {
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      UXMode,
				},
				LoginState: sso.LoginState{
					MergeRealm:      loginid.DefaultRealm,
					OnUserDuplicate: model.OnUserDuplicateAbort,
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)

			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}
			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()
			sh.ServeHTTP(resp, req)

			So(resp.Code, ShouldEqual, 302)

			actual, err := decodeResultInURL(resp.Result().Header.Get("Location"))
			So(err, ShouldBeNil)
			So(actual, ShouldEqualJSON, `
			{
				"callback_url": "http://localhost:3000",
				"result": {
					"error": {
						"name": "AlreadyExists",
						"reason": "LoginIDAlreadyUsed",
						"message": "login ID is already used",
						"code": 409
					}
				}
			}
			`)
		})

		Convey("OnUserDuplicate == merge", func() {
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      UXMode,
				},
				LoginState: sso.LoginState{
					MergeRealm:      loginid.DefaultRealm,
					OnUserDuplicate: model.OnUserDuplicateMerge,
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)

			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}
			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()
			sh.ServeHTTP(resp, req)

			So(resp.Code, ShouldEqual, 302)

			actual, err := decodeResultInURL(resp.Result().Header.Get("Location"))
			So(err, ShouldBeNil)
			p, _ := sh.OAuthAuthProvider.GetPrincipalByProvider(oauth.GetByProviderOptions{
				ProviderType:   "google",
				ProviderUserID: providerUserID,
			})
			So(actual, ShouldEqualJSON, fmt.Sprintf(`
			{
				"callback_url": "http://localhost:3000",
				"result": {
					"result": {
						"action": "login",
						"code_challenge": "",
						"session_create_reason": "login",
						"user_id": "%s",
						"principal_id": "%s"
					}
				}
			}
			`, p.UserID, p.ID))
		})

		Convey("OnUserDuplicate == create", func() {
			state := sso.State{
				Action: action,
				OAuthAuthorizationCodeFlowState: sso.OAuthAuthorizationCodeFlowState{
					CallbackURL: "http://localhost:3000",
					UXMode:      UXMode,
				},
				LoginState: sso.LoginState{
					MergeRealm:      loginid.DefaultRealm,
					OnUserDuplicate: model.OnUserDuplicateCreate,
				},
				Nonce: hashedNonce,
			}
			encodedState, _ := mockProvider.EncodeState(state)

			v := url.Values{}
			v.Set("code", "code")
			v.Add("state", encodedState)
			u := url.URL{
				RawQuery: v.Encode(),
			}
			req, _ := http.NewRequest("GET", u.RequestURI(), nil)
			req.AddCookie(nonceCookie)
			resp := httptest.NewRecorder()
			sh.ServeHTTP(resp, req)

			So(resp.Code, ShouldEqual, 302)

			actual, err := decodeResultInURL(resp.Result().Header.Get("Location"))
			So(err, ShouldBeNil)
			p, _ := sh.OAuthAuthProvider.GetPrincipalByProvider(oauth.GetByProviderOptions{
				ProviderType:   "google",
				ProviderUserID: providerUserID,
			})
			So(p.UserID, ShouldNotEqual, "john.doe.id")
			So(actual, ShouldEqualJSON, fmt.Sprintf(`
			{
				"callback_url": "http://localhost:3000",
				"result": {
					"result": {
						"action": "login",
						"code_challenge": "",
						"session_create_reason": "signup",
						"user_id": "%s",
						"principal_id": "%s"
					}
				}
			}
			`, p.UserID, p.ID))
		})
	})
}
