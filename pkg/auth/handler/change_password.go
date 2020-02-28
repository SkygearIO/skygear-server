package handler

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/core/validation"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/auth/task"
	"github.com/skygeario/skygear-server/pkg/core/async"
	"github.com/skygeario/skygear-server/pkg/core/audit"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/event"
	"github.com/skygeario/skygear-server/pkg/core/auth/hook"
	coreAuthModel "github.com/skygeario/skygear-server/pkg/core/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/passwordpolicy"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal/password"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/skyerr"
)

func AttachChangePasswordHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/change_password", &ChangePasswordHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

// ChangePasswordHandlerFactory creates ChangePasswordHandler
type ChangePasswordHandlerFactory struct {
	Dependency auth.DependencyMap
}

// NewHandler creates new handler
func (f ChangePasswordHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &ChangePasswordHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	return h.RequireAuthz(h, h)
}

type ChangePasswordRequestPayload struct {
	NewPassword string `json:"password"`
	OldPassword string `json:"old_password"`
}

// nolint:gosec
// @JSONSchema
const ChangePasswordRequestSchema = `
{
	"$id": "#ChangePasswordRequest",
	"type": "object",
	"properties": {
		"password": { "type": "string", "minLength": 1 },
		"old_password": { "type": "string", "minLength": 1 }
	},
	"required": ["password", "old_password"]
}
`

/*
	@Operation POST /change_password - Change password
		Changes current user password.

		@Tag User
		@SecurityRequirement access_key
		@SecurityRequirement access_token

		@RequestBody
			Describe old and new password.
			@JSONSchema {ChangePasswordRequest}

		@Response 200
			Return user and new access token.
			@JSONSchema {AuthResponse}

		@Callback password_update {PasswordUpdateEvent}
		@Callback user_sync {UserSyncEvent}
*/
type ChangePasswordHandler struct {
	Validator            *validation.Validator           `dependency:"Validator"`
	AuditTrail           audit.Trail                     `dependency:"AuditTrail"`
	AuthContext          coreAuth.ContextGetter          `dependency:"AuthContextGetter"`
	RequireAuthz         handler.RequireAuthz            `dependency:"RequireAuthz"`
	AuthInfoStore        authinfo.Store                  `dependency:"AuthInfoStore"`
	PasswordAuthProvider password.Provider               `dependency:"PasswordAuthProvider"`
	IdentityProvider     principal.IdentityProvider      `dependency:"IdentityProvider"`
	PasswordChecker      *passwordpolicy.PasswordChecker `dependency:"PasswordChecker"`
	SessionProvider      session.Provider                `dependency:"SessionProvider"`
	SessionWriter        session.Writer                  `dependency:"SessionWriter"`
	TxContext            db.TxContext                    `dependency:"TxContext"`
	UserProfileStore     userprofile.Store               `dependency:"UserProfileStore"`
	HookProvider         hook.Provider                   `dependency:"HookProvider"`
	TaskQueue            async.Queue                     `dependency:"AsyncTaskQueue"`
}

// ProvideAuthzPolicy provides authorization policy of handler
func (h ChangePasswordHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		policy.RequireValidUser,
	)
}

func (h ChangePasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error

	result, err := h.Handle(w, r)
	if err == nil {
		h.SessionWriter.WriteSession(w, &result.AccessToken, nil)
		handler.WriteResponse(w, handler.APIResponse{Result: result})
	} else {
		handler.WriteResponse(w, handler.APIResponse{Error: err})
	}
}

func (h ChangePasswordHandler) Handle(w http.ResponseWriter, r *http.Request) (resp model.AuthResponse, err error) {
	var payload ChangePasswordRequestPayload
	if err = handler.BindJSONBody(r, w, h.Validator, "#ChangePasswordRequest", &payload); err != nil {
		return
	}

	err = hook.WithTx(h.HookProvider, h.TxContext, func() error {
		authinfo, _ := h.AuthContext.AuthInfo()
		sess, _ := h.AuthContext.Session()

		if err := h.PasswordChecker.ValidatePassword(passwordpolicy.ValidatePasswordPayload{
			PlainPassword: payload.NewPassword,
			AuthID:        authinfo.ID,
		}); err != nil {
			return err
		}

		principals, err := h.PasswordAuthProvider.GetPrincipalsByUserID(authinfo.ID)
		if err != nil {
			return err
		}
		if len(principals) == 0 {
			err = skyerr.NewInvalid("user has no password")
			return err
		}

		principal := principals[0]
		for _, p := range principals {
			if p.ID == sess.PrincipalID {
				principal = p
			}
			err = p.VerifyPassword(payload.OldPassword)
			if err != nil {
				return err
			}
			err = h.PasswordAuthProvider.UpdatePassword(p, payload.NewPassword)
			if err != nil {
				return err
			}
		}

		// refresh session
		accessToken, err := h.SessionProvider.Refresh(sess)
		if err != nil {
			return err
		}
		tokens := coreAuth.SessionTokens{ID: sess.ID, AccessToken: accessToken}

		// Get Profile
		userProfile, err := h.UserProfileStore.GetUserProfile(authinfo.ID)
		if err != nil {
			return err
		}

		user := coreAuthModel.NewUser(*authinfo, userProfile)
		identity := coreAuthModel.NewIdentity(principal)

		err = h.HookProvider.DispatchEvent(
			event.PasswordUpdateEvent{
				Reason: event.PasswordUpdateReasonChangePassword,
				User:   user,
			},
			&user,
		)
		if err != nil {
			return err
		}

		resp = model.NewAuthResponse(user, identity, tokens, "")

		h.AuditTrail.Log(audit.Entry{
			UserID: authinfo.ID,
			Event:  audit.EventChangePassword,
		})

		// password house keeper
		h.TaskQueue.Enqueue(task.PwHousekeeperTaskName, task.PwHousekeeperTaskParam{
			AuthID: authinfo.ID,
		}, nil)

		return nil
	})
	return
}
