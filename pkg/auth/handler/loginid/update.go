package loginid

import (
	"errors"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userverify"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/event"
	"github.com/skygeario/skygear-server/pkg/core/auth/hook"
	coreAuthModel "github.com/skygeario/skygear-server/pkg/core/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal/password"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/loginid"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

func AttachUpdateLoginIDHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/login_id/update", &UpdateLoginIDHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type UpdateLoginIDHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f UpdateLoginIDHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &UpdateLoginIDHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	return h.RequireAuthz(h, h)
}

type UpdateLoginIDRequestPayload struct {
	OldLoginID loginid.LoginID `json:"old_login_id"`
	NewLoginID loginid.LoginID `json:"new_login_id"`
}

// @JSONSchema
const UpdateLoginIDRequestSchema = `
{
	"$id": "#UpdateLoginIDRequest",
	"type": "object",
	"properties": {
		"old_login_id": {
			"type": "object",
			"properties": {
				"key": { "type": "string", "minLength": 1 },
				"value": { "type": "string", "minLength": 1 }
			},
			"required": ["key", "value"]
		},
		"new_login_id": {
			"type": "object",
			"properties": {
				"key": { "type": "string", "minLength": 1 },
				"value": { "type": "string", "minLength": 1 }
			},
			"required": ["key", "value"]
		}
	},
	"required": ["old_login_id", "new_login_id"]
}
`

/*
	@Operation POST /login_id/update - update login ID
		Update the specified login ID for current user.
		This operation is same as adding the new login ID and then deleting
		old login ID atomically.

		@Tag User
		@SecurityRequirement access_key
		@SecurityRequirement access_token

		@RequestBody
			Describe the new login ID.
			@JSONSchema {UpdateLoginIDRequest}

		@Response 200
			Updated user and identity info.
			@JSONSchema {UserIdentityResponse}

		@Callback identity_create {UserSyncEvent}
		@Callback identity_delete {UserSyncEvent}
		@Callback user_sync {UserSyncEvent}
*/
type UpdateLoginIDHandler struct {
	Validator                *validation.Validator      `dependency:"Validator"`
	AuthContext              coreAuth.ContextGetter     `dependency:"AuthContextGetter"`
	RequireAuthz             handler.RequireAuthz       `dependency:"RequireAuthz"`
	AuthInfoStore            authinfo.Store             `dependency:"AuthInfoStore"`
	PasswordAuthProvider     password.Provider          `dependency:"PasswordAuthProvider"`
	IdentityProvider         principal.IdentityProvider `dependency:"IdentityProvider"`
	UserVerificationProvider userverify.Provider        `dependency:"UserVerificationProvider"`
	SessionProvider          session.Provider           `dependency:"SessionProvider"`
	TxContext                db.TxContext               `dependency:"TxContext"`
	UserProfileStore         userprofile.Store          `dependency:"UserProfileStore"`
	HookProvider             hook.Provider              `dependency:"HookProvider"`
	Logger                   *logrus.Entry              `dependency:"HandlerLogger"`
}

func (h UpdateLoginIDHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		policy.RequireValidUser,
	)
}

func (h UpdateLoginIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Handle(w, r)
	if err == nil {
		handler.WriteResponse(w, handler.APIResponse{Result: resp})
	} else {
		handler.WriteResponse(w, handler.APIResponse{Error: err})
	}
}

func (h UpdateLoginIDHandler) Handle(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	var payload UpdateLoginIDRequestPayload
	if err := handler.BindJSONBody(r, w, h.Validator, "#UpdateLoginIDRequest", &payload); err != nil {
		return nil, err
	}

	var resp interface{}
	err := hook.WithTx(h.HookProvider, h.TxContext, func() error {
		authInfo, _ := h.AuthContext.AuthInfo()
		userID := authInfo.ID

		var oldPrincipal password.Principal
		err := h.PasswordAuthProvider.GetPrincipalByLoginIDWithRealm(
			payload.OldLoginID.Key,
			payload.OldLoginID.Value,
			loginid.DefaultRealm,
			&oldPrincipal,
		)
		if err != nil {
			if errors.Is(err, principal.ErrNotFound) {
				err = password.ErrLoginIDNotFound
			}
			return err
		}
		if oldPrincipal.UserID != userID {
			return password.ErrLoginIDNotFound
		}

		newPrincipal, err := h.PasswordAuthProvider.MakePrincipal(userID, "", payload.NewLoginID, loginid.DefaultRealm)
		if err != nil {
			return err
		}
		newPrincipal.HashedPassword = oldPrincipal.HashedPassword

		err = h.PasswordAuthProvider.CreatePrincipal(newPrincipal)
		if err != nil {
			return err
		}

		err = h.PasswordAuthProvider.DeletePrincipal(&oldPrincipal)
		if err != nil {
			return err
		}

		principals, err := h.PasswordAuthProvider.GetPrincipalsByUserID(userID)
		if err != nil {
			return err
		}
		newPrincipalIndex := -1
		for i, principal := range principals {
			if principal.ID == newPrincipal.ID {
				newPrincipalIndex = i
				break
			}
		}
		if newPrincipalIndex == -1 {
			panic("login_id_update: cannot find new principal")
		}
		err = validateLoginIDs(h.PasswordAuthProvider, extractLoginIDs(principals), newPrincipalIndex)
		if err != nil {
			return err
		}

		var userProfile userprofile.UserProfile
		userProfile, err = h.UserProfileStore.GetUserProfile(userID)
		if err != nil {
			return err
		}
		user := coreAuthModel.NewUser(*authInfo, userProfile)

		delete(authInfo.VerifyInfo, oldPrincipal.LoginID)
		err = h.UserVerificationProvider.UpdateVerificationState(authInfo, h.AuthInfoStore, principals)
		if err != nil {
			return err
		}

		newIdentity := coreAuthModel.NewIdentity(newPrincipal)
		err = h.HookProvider.DispatchEvent(
			event.IdentityCreateEvent{
				User:     user,
				Identity: newIdentity,
			},
			&user,
		)
		if err != nil {
			return err
		}
		oldIdentity := coreAuthModel.NewIdentity(&oldPrincipal)
		err = h.HookProvider.DispatchEvent(
			event.IdentityDeleteEvent{
				User:     user,
				Identity: oldIdentity,
			},
			&user,
		)
		if err != nil {
			return err
		}

		sessions, err := h.SessionProvider.List(userID)
		if err != nil {
			return err
		}

		for _, session := range sessions {
			if session.PrincipalID == oldPrincipal.ID {
				err = h.SessionProvider.UpdatePrincipal(session, newPrincipal.ID)
				if err != nil {
					// log and ignore error
					h.Logger.WithError(err).Error("Cannot update session principal ID")
				}
			}
		}

		user = coreAuthModel.NewUser(*authInfo, userProfile)
		resp = model.NewAuthResponseWithUserIdentity(user, newIdentity)
		return nil
	})
	return resp, err
}
