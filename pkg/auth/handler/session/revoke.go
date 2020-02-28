package session

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/auth"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/event"
	"github.com/skygeario/skygear-server/pkg/core/auth/hook"
	"github.com/skygeario/skygear-server/pkg/core/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/model/format"
	"github.com/skygeario/skygear-server/pkg/core/auth/principal"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	"github.com/skygeario/skygear-server/pkg/core/auth/userprofile"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

func AttachRevokeHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/session/revoke", &RevokeHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type RevokeHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f RevokeHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &RevokeHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	return h.RequireAuthz(h, h)
}

type RevokeRequestPayload struct {
	CurrentSessionID string `json:"-"`
	SessionID        string `json:"session_id"`
}

func (p *RevokeRequestPayload) Validate() []validation.ErrorCause {
	if p.CurrentSessionID != p.SessionID {
		return nil
	}
	return []validation.ErrorCause{{
		Kind:    validation.ErrorGeneral,
		Pointer: "/session_id",
		Message: "session_id must not be current session",
	}}
}

// @JSONSchema
const RevokeRequestSchema = `
{
	"$id": "#SessionRevokeRequest",
	"type": "object",
	"properties": {
		"session_id": { "type": "string", "minLength": 1 }
	},
	"required": ["session_id"]
}
`

/*
	@Operation POST /session/revoke - Revoke session
		Update specified session. Current session cannot be revoked.

		@Tag User
		@SecurityRequirement access_key
		@SecurityRequirement access_token

		@RequestBody
			Describe the session ID.
			@JSONSchema {SessionRevokeRequest}

		@Response 200 {EmptyResponse}
*/
type RevokeHandler struct {
	AuthContext      coreAuth.ContextGetter     `dependency:"AuthContextGetter"`
	Validator        *validation.Validator      `dependency:"Validator"`
	RequireAuthz     handler.RequireAuthz       `dependency:"RequireAuthz"`
	TxContext        db.TxContext               `dependency:"TxContext"`
	SessionProvider  session.Provider           `dependency:"SessionProvider"`
	IdentityProvider principal.IdentityProvider `dependency:"IdentityProvider"`
	UserProfileStore userprofile.Store          `dependency:"UserProfileStore"`
	HookProvider     hook.Provider              `dependency:"HookProvider"`
}

func (h RevokeHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		policy.RequireValidUser,
	)
}

func (h RevokeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var response handler.APIResponse
	var payload RevokeRequestPayload
	session, _ := h.AuthContext.Session()
	payload.CurrentSessionID = session.ID
	if err := handler.BindJSONBody(r, w, h.Validator, "#SessionRevokeRequest", &payload); err != nil {
		response.Error = err
	} else {
		result, err := h.Handle(payload)
		if err != nil {
			response.Error = err
		} else {
			response.Result = result
		}
	}
	handler.WriteResponse(w, response)
}

func (h RevokeHandler) Handle(payload RevokeRequestPayload) (resp interface{}, err error) {
	err = db.WithTx(h.TxContext, func() error {
		authInfo, _ := h.AuthContext.AuthInfo()
		userID := authInfo.ID
		sessionID := payload.SessionID

		// ignore session not found errors
		s, err := h.SessionProvider.Get(sessionID)
		if err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				err = nil
				resp = map[string]string{}
			}
			return err
		}
		if s.UserID != userID {
			resp = map[string]string{}
			return err
		}

		var profile userprofile.UserProfile
		if profile, err = h.UserProfileStore.GetUserProfile(s.UserID); err != nil {
			return err
		}

		var principal principal.Principal
		if principal, err = h.IdentityProvider.GetPrincipalByID(s.PrincipalID); err != nil {
			return err
		}

		user := model.NewUser(*authInfo, profile)
		identity := model.NewIdentity(principal)
		session := format.SessionFromSession(s)

		err = h.HookProvider.DispatchEvent(
			event.SessionDeleteEvent{
				Reason:   event.SessionDeleteReasonRevoke,
				User:     user,
				Identity: identity,
				Session:  session,
			},
			&user,
		)
		if err != nil {
			return err
		}

		err = h.SessionProvider.Invalidate(s)
		if err != nil {
			return err
		}

		resp = struct{}{}
		return nil
	})
	return
}
