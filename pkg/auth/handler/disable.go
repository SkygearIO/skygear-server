package handler

import (
	"net/http"
	"time"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/hook"
	"github.com/skygeario/skygear-server/pkg/auth/event"
	authModel "github.com/skygeario/skygear-server/pkg/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/audit"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

// AttachSetDisableHandler attaches SetDisableHandler to server
func AttachSetDisableHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/disable/set", &SetDisableHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

// SetDisableHandlerFactory creates SetDisableHandler
type SetDisableHandlerFactory struct {
	Dependency auth.DependencyMap
}

// NewHandler creates new SetDisableHandler
func (f SetDisableHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &SetDisableHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	h.AuditTrail = h.AuditTrail.WithRequest(request)
	return h.RequireAuthz(h, h)
}

type setDisableUserPayload struct {
	UserID       string `json:"user_id"`
	Disabled     bool   `json:"disabled"`
	Message      string `json:"message"`
	ExpiryString string `json:"expiry"`
	expiry       *time.Time
}

func (p *setDisableUserPayload) SetDefaultValue() {
	if p.ExpiryString != "" {
		expiry, err := time.Parse(time.RFC3339, p.ExpiryString)
		if err != nil {
			// should be already validated, so panic if unexpected.
			panic(err)
		}
		p.expiry = &expiry
	}
}

// @JSONSchema
const SetDisableRequestSchema = `
{
	"$id": "#SetDisableRequest",
	"type": "object",
	"properties": {
		"user_id": { "type": "string", "minLength": 1 },
		"disabled": { "type": "boolean" },
		"message": { "type": "string", "minLength": 1 },
		"expiry": { "type": "string", "format": "date-time" }
	},
	"required": ["user_id", "disabled"]
}
`

/*
	@Operation POST /disable/set - Set user disabled status
		Disable/enable target user.

		@Tag Administration
		@SecurityRequirement master_key
		@SecurityRequirement access_token

		@RequestBody
			Describe target user and desired disable status.
			@JSONSchema {SetDisableRequest}
			@JSONExample EnableUser - Enable user
				{
					"auth_id": "F1D4AAAC-A31A-4471-92B2-6E08376BDD87",
					"disabled": false
				}
			@JSONExample DisableUser - Disable user permanently
				{
					"auth_id": "F1D4AAAC-A31A-4471-92B2-6E08376BDD87",
					"disabled": true
				}
			@JSONExample DisableUserExpiry - Disable user with expiry
				{
					"auth_id": "F1D4AAAC-A31A-4471-92B2-6E08376BDD87",
					"disabled": true,
					"message": "Banned",
					"expiry": "2019-07-31T09:39:22.349Z"
				}

		@Response 200 {EmptyResponse}

		@Callback user_update {UserUpdateEvent}
		@Callback user_sync {UserSyncEvent}
*/
type SetDisableHandler struct {
	Validator        *validation.Validator  `dependency:"Validator"`
	AuthContext      coreAuth.ContextGetter `dependency:"AuthContextGetter"`
	RequireAuthz     handler.RequireAuthz   `dependency:"RequireAuthz"`
	AuthInfoStore    authinfo.Store         `dependency:"AuthInfoStore"`
	UserProfileStore userprofile.Store      `dependency:"UserProfileStore"`
	AuditTrail       audit.Trail            `dependency:"AuditTrail"`
	HookProvider     hook.Provider          `dependency:"HookProvider"`
	TxContext        db.TxContext           `dependency:"TxContext"`
}

// ProvideAuthzPolicy provides authorization policy of handler
func (h SetDisableHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		authz.PolicyFunc(policy.RequireMasterKey),
	)
}

func (h SetDisableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	result, err := h.Handle(w, r)
	if err == nil {
		handler.WriteResponse(w, handler.APIResponse{Result: result})
	} else {
		handler.WriteResponse(w, handler.APIResponse{Error: err})
	}
}

func (h SetDisableHandler) Handle(w http.ResponseWriter, r *http.Request) (resp interface{}, err error) {
	var payload setDisableUserPayload
	if err = handler.BindJSONBody(r, w, h.Validator, "#SetDisableRequest", &payload); err != nil {
		return
	}

	err = hook.WithTx(h.HookProvider, h.TxContext, func() error {
		info := authinfo.AuthInfo{}
		if err = h.AuthInfoStore.GetAuth(payload.UserID, &info); err != nil {
			return err
		}

		profile, err := h.UserProfileStore.GetUserProfile(info.ID)
		if err != nil {
			return err
		}

		oldUser := authModel.NewUser(info, profile)

		info.Disabled = payload.Disabled
		if !info.Disabled {
			info.DisabledMessage = ""
			info.DisabledExpiry = nil
		} else {
			info.DisabledMessage = payload.Message
			info.DisabledExpiry = payload.expiry
		}

		if err = h.AuthInfoStore.UpdateAuth(&info); err != nil {
			return err
		}

		user := authModel.NewUser(info, profile)

		err = h.HookProvider.DispatchEvent(
			event.UserUpdateEvent{
				Reason:     event.UserUpdateReasonAdministrative,
				User:       oldUser,
				IsDisabled: &payload.Disabled,
			},
			&user,
		)
		if err != nil {
			return err
		}

		h.logAuditTrail(payload)

		resp = struct{}{}
		return nil
	})
	return
}

func (h SetDisableHandler) logAuditTrail(p setDisableUserPayload) {
	var event audit.Event
	if p.Disabled {
		event = audit.EventDisableUser
	} else {
		event = audit.EventEnableUser
	}

	h.AuditTrail.Log(audit.Entry{
		UserID: p.UserID,
		Event:  event,
	})
}
