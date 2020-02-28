package loginid

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	. "github.com/skygeario/skygear-server/pkg/core/skytest"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/principal/password"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userverify"
	"github.com/skygeario/skygear-server/pkg/auth/event"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/metadata"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	authtest "github.com/skygeario/skygear-server/pkg/core/auth/testing"
	"github.com/skygeario/skygear-server/pkg/core/config"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

func TestUpdateLoginIDHandler(t *testing.T) {
	Convey("Test UpdateLoginIDHandler", t, func() {
		h := &UpdateLoginIDHandler{}
		validator := validation.NewValidator("http://v2.skygear.io")
		validator.AddSchemaFragments(
			UpdateLoginIDRequestSchema,
		)
		h.Validator = validator
		h.TxContext = db.NewMockTxContext()
		authContext := authtest.NewMockContext().
			UseUser("user-id-1", "principal-id-1").
			SetVerifyInfo(map[string]bool{"user1@example.com": true}).
			MarkVerified()
		h.AuthContext = authContext
		authInfoStore := authinfo.NewMockStoreWithAuthInfoMap(
			map[string]authinfo.AuthInfo{
				"user-id-1": *authContext.MustAuthInfo(),
			},
		)
		h.AuthInfoStore = authInfoStore
		passwordAuthProvider := password.NewMockProviderWithPrincipalMap(
			[]config.LoginIDKeyConfiguration{
				newLoginIDKeyConfig("email", config.LoginIDKeyType(metadata.Email), 1),
				newLoginIDKeyConfig("username", config.LoginIDKeyType(metadata.Username), 1),
			},
			[]string{loginid.DefaultRealm},
			map[string]password.Principal{
				"principal-id-1": password.Principal{
					ID:         "principal-id-1",
					UserID:     "user-id-1",
					LoginIDKey: "email",
					LoginID:    "user1@example.com",
					Realm:      loginid.DefaultRealm,
					ClaimsValue: map[string]interface{}{
						"email": "user1@example.com",
					},
				},
				"principal-id-2": password.Principal{
					ID:         "principal-id-2",
					UserID:     "user-id-1",
					LoginIDKey: "username",
					LoginID:    "user1",
					Realm:      loginid.DefaultRealm,
					ClaimsValue: map[string]interface{}{
						"username": "user1",
					},
				},
				"principal-id-3": password.Principal{
					ID:         "principal-id-3",
					UserID:     "user-id-2",
					LoginIDKey: "username",
					LoginID:    "user2",
					Realm:      loginid.DefaultRealm,
					ClaimsValue: map[string]interface{}{
						"username": "user2",
					},
				},
			},
		)
		h.PasswordAuthProvider = passwordAuthProvider
		h.IdentityProvider = principal.NewMockIdentityProvider(passwordAuthProvider)
		sessionProvider := session.NewMockProvider()
		sessionProvider.Sessions["session-id"] = auth.Session{
			ID:          "session-id",
			ClientID:    "web-app",
			UserID:      "user-id-1",
			PrincipalID: "principal-id-1",
		}
		h.SessionProvider = sessionProvider
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
				"old_login_id": { "key": "username", "value": "user" },
				"new_login_id": { "key": "username", "value": "user1_a" }
			}`))
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
				"old_login_id": { "key": "username", "value": "user2" },
				"new_login_id": { "key": "username", "value": "user1_a" }
			}`))
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

		Convey("should fail if login ID is already used", func() {
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"old_login_id": { "key": "username", "value": "user1" },
				"new_login_id": { "key": "username", "value": "user2" }
			}`))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Body.Bytes(), ShouldEqualJSON, `{
				"error": {
					"name": "AlreadyExists",
					"reason": "LoginIDAlreadyUsed",
					"message": "login ID is already used",
					"code": 409
				}
			}`)
		})

		Convey("should fail if login ID amount is invalid", func() {
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"old_login_id": { "key": "email", "value": "user1@example.com" },
				"new_login_id": { "key": "username", "value": "user1_a" }
			}`))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Body.Bytes(), ShouldEqualJSON, `{
				"error": {
					"name": "Invalid",
					"reason": "ValidationFailed",
					"message": "invalid login ID",
					"code": 400,
					"info": {
						"causes": [
							{
								"kind": "EntryAmount",
								"message": "too many login IDs",
								"pointer": "",
								"details": { "key": "username", "lte": 1 }
							}
						]
					}
				}
			}`)
		})

		Convey("should update login ID", func() {
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"old_login_id": { "key": "email", "value": "user1@example.com" },
				"new_login_id": { "key": "email", "value": "user1+a@example.com" }
			}`))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 200)

			So(passwordAuthProvider.PrincipalMap, ShouldHaveLength, 3)
			var p password.Principal
			err := passwordAuthProvider.GetPrincipalByLoginIDWithRealm("email", "user1@example.com", loginid.DefaultRealm, &p)
			So(err, ShouldBeError, "principal not found")
			err = passwordAuthProvider.GetPrincipalByLoginIDWithRealm("email", "user1+a@example.com", loginid.DefaultRealm, &p)
			So(err, ShouldBeNil)
			So(p.UserID, ShouldEqual, "user-id-1")
			So(p.LoginIDKey, ShouldEqual, "email")
			So(p.LoginID, ShouldEqual, "user1+a@example.com")

			So(w.Body.Bytes(), ShouldEqualJSON, fmt.Sprintf(`{
				"result": {
					"user": {
						"id": "user-id-1",
						"created_at": "0001-01-01T00:00:00Z",
						"is_disabled": false,
						"is_manually_verified": false,
						"is_verified": false,
						"verify_info": {},
						"metadata": {}
					},
					"identity": {
						"id": "%s",
						"type": "password",
						"login_id": "user1+a@example.com",
						"login_id_key": "email",
						"claims": {
							"email": "user1+a@example.com"
						}
					}
				}
			}`, p.ID))

			So(sessionProvider.Sessions, ShouldHaveLength, 1)
			So(sessionProvider.Sessions["session-id"].PrincipalID, ShouldEqual, p.ID)

			So(hookProvider.DispatchedEvents, ShouldResemble, []event.Payload{
				event.IdentityCreateEvent{
					User: model.User{
						ID:         "user-id-1",
						Verified:   true,
						Disabled:   false,
						VerifyInfo: map[string]bool{"user1@example.com": true},
						Metadata:   userprofile.Data{},
					},
					Identity: model.Identity{
						ID:   p.ID,
						Type: "password",
						Attributes: principal.Attributes{
							"login_id_key": "email",
							"login_id":     "user1+a@example.com",
						},
						Claims: principal.Claims{
							"email": "user1+a@example.com",
						},
					},
				},
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
			r, _ := http.NewRequest("POST", "", strings.NewReader(`{
				"old_login_id": { "key": "email", "value": "user1@example.com" },
				"new_login_id": { "key": "email", "value": "user1+a@example.com" }
			}`))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)

			So(w.Code, ShouldEqual, 200)

			So(authInfoStore.AuthInfoMap["user-id-1"].VerifyInfo["user1@example.com"], ShouldBeFalse)
			So(authInfoStore.AuthInfoMap["user-id-1"].Verified, ShouldBeFalse)
		})
	})
}
