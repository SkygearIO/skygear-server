package loginid

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/skygeario/skygear-server/pkg/core/skytest"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/auth"
	authtesting "github.com/skygeario/skygear-server/pkg/auth/dependency/auth/testing"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/password"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userverify"
	"github.com/skygeario/skygear-server/pkg/auth/event"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/metadata"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

type mockRemoveSessionManager struct {
	Sessions []auth.AuthSession
}

func (m *mockRemoveSessionManager) List(userID string) ([]auth.AuthSession, error) {
	return m.Sessions, nil
}

func (m *mockRemoveSessionManager) Revoke(s auth.AuthSession) error {
	n := 0
	for _, session := range m.Sessions {
		if session.SessionID() == s.SessionID() {
			continue
		}
		m.Sessions[n] = session
		n++
	}
	m.Sessions = m.Sessions[:n]
	return nil
}

func TestRemoveLoginIDHandler(t *testing.T) {
	Convey("Test RemoveLoginIDHandler", t, func() {
		h := &RemoveLoginIDHandler{}
		validator := validation.NewValidator("http://v2.skygear.io")
		validator.AddSchemaFragments(
			RemoveLoginIDRequestSchema,
		)
		h.Validator = validator
		h.TxContext = db.NewMockTxContext()
		authctx := authtesting.WithAuthn().
			UserID("user-id-1").
			PrincipalID("principal-id-1").
			Verified(true)
		authInfoStore := authinfo.NewMockStoreWithAuthInfoMap(
			map[string]authinfo.AuthInfo{
				"user-id-1": authinfo.AuthInfo{
					ID:         "user-id-1",
					Verified:   true,
					VerifyInfo: map[string]bool{"user1@example.com": true},
				},
			},
		)
		h.AuthInfoStore = authInfoStore
		passwordAuthProvider := password.NewMockProviderWithPrincipalMap(
			[]config.LoginIDKeyConfiguration{
				newLoginIDKeyConfig("email", config.LoginIDKeyType(metadata.Email), 1),
				newLoginIDKeyConfig("username", config.LoginIDKeyType(metadata.Username), 1),
			},
			[]string{password.DefaultRealm},
			map[string]password.Principal{
				"principal-id-1": password.Principal{
					ID:         "principal-id-1",
					UserID:     "user-id-1",
					LoginIDKey: "email",
					LoginID:    "user1@example.com",
					Realm:      password.DefaultRealm,
					ClaimsValue: map[string]interface{}{
						"email": "user1@example.com",
					},
				},
				"principal-id-2": password.Principal{
					ID:         "principal-id-2",
					UserID:     "user-id-1",
					LoginIDKey: "username",
					LoginID:    "user1",
					Realm:      password.DefaultRealm,
					ClaimsValue: map[string]interface{}{
						"username": "user1",
					},
				},
				"principal-id-3": password.Principal{
					ID:         "principal-id-3",
					UserID:     "user-id-2",
					LoginIDKey: "username",
					LoginID:    "user2",
					Realm:      password.DefaultRealm,
					ClaimsValue: map[string]interface{}{
						"username": "user2",
					},
				},
			},
		)
		h.PasswordAuthProvider = passwordAuthProvider
		sessionManager := &mockRemoveSessionManager{}
		sessionManager.Sessions = []auth.AuthSession{
			authtesting.WithAuthn().
				SessionID("session-id").
				UserID("user-id-1").
				PrincipalID("principal-id-1").
				ToSession(),
		}
		h.SessionManager = sessionManager
		h.UserVerificationProvider = userverify.NewProvider(nil, nil, &config.UserVerificationConfiguration{
			Criteria: config.UserVerificationCriteriaAll,
			LoginIDKeys: []config.UserVerificationKeyConfiguration{
				config.UserVerificationKeyConfiguration{Key: "email"},
			},
		}, nil)
		h.UserProfileStore = userprofile.NewMockUserProfileStore()
		hookProvider := hook.NewMockProvider()
		h.HookProvider = hookProvider

		Convey("should fail if login ID does not exist", func() {
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"key": "username", "value": "user"
			}`))
			r = authctx.ToRequest(r)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Body.Bytes(), ShouldEqualJSON, `{
				"error": {
					"name": "NotFound",
					"reason": "LoginIDNotFound",
					"message": "login ID does not exist",
					"code": 404
				}
			}`)
		})

		Convey("should fail if login ID does not belong to the user", func() {
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"key": "username", "value": "user2"
			}`))
			r = authctx.ToRequest(r)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Body.Bytes(), ShouldEqualJSON, `{
				"error": {
					"name": "NotFound",
					"reason": "LoginIDNotFound",
					"message": "login ID does not exist",
					"code": 404
				}
			}`)
		})

		Convey("should fail if attempted to delete current identity", func() {
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"key": "email", "value": "user1@example.com"
			}`))
			r = authctx.ToRequest(r)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Body.Bytes(), ShouldEqualJSON, `{
				"error": {
					"name": "Invalid",
					"reason": "CurrentIdentityBeingDeleted",
					"message": "must not delete current identity",
					"code": 400
				}
			}`)
		})

		Convey("should remove login ID", func() {
			authctx = authctx.PrincipalID("principal-id-2")
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"key": "email", "value": "user1@example.com"
			}`))
			r = authctx.ToRequest(r)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Body.Bytes(), ShouldEqualJSON, `{
				"result": {}
			}`)

			So(passwordAuthProvider.PrincipalMap, ShouldHaveLength, 2)
			_, err := passwordAuthProvider.GetPrincipalByLoginID("email", "user1@example.com")
			So(err, ShouldBeError, "principal not found")

			So(sessionManager.Sessions, ShouldBeEmpty)

			So(hookProvider.DispatchedEvents, ShouldResemble, []event.Payload{
				event.IdentityDeleteEvent{
					User: model.User{
						ID:         "user-id-1",
						Verified:   true,
						Disabled:   false,
						VerifyInfo: map[string]bool{"user1@example.com": true},
						Metadata:   userprofile.Data{},
					},
					Identity: model.Identity{
						ID:   "principal-id-1",
						Type: "password",
						Attributes: principal.Attributes{
							"login_id_key": "email",
							"login_id":     "user1@example.com",
						},
						Claims: principal.Claims{
							"email": "user1@example.com",
						},
					},
				},
			})
		})

		Convey("should invalidate verify state", func() {
			authctx = authctx.PrincipalID("principal-id-2")
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"key": "email", "value": "user1@example.com"
			}`))
			r = authctx.ToRequest(r)
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Body.Bytes(), ShouldEqualJSON, `{
				"result": {}
			}`)

			So(authInfoStore.AuthInfoMap["user-id-1"].VerifyInfo["user1@example.com"], ShouldBeFalse)
			So(authInfoStore.AuthInfoMap["user-id-1"].Verified, ShouldBeFalse)
		})
	})
}
