package session

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/auth"
	authSession "github.com/skygeario/skygear-server/pkg/auth/dependency/session"
	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/model"
	"github.com/skygeario/skygear-server/pkg/core/auth/session"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/errors"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/core/validation"
)

func AttachGetHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/session/get", &GetHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

type GetHandlerFactory struct {
	Dependency auth.DependencyMap
}

func (f GetHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &GetHandler{}
	inject.DefaultRequestInject(h, f.Dependency, request)
	return h.RequireAuthz(h, h)
}

type GetRequestPayload struct {
	SessionID string `json:"session_id"`
}

// @JSONSchema
const GetRequestSchema = `
{
	"$id": "#SessionGetRequest",
	"type": "object",
	"properties": {
		"session_id": { "type": "string", "minLength": 1 }
	},
	"required": ["session_id"]
}
`

type GetResponse struct {
	Session model.Session `json:"session"`
}

// @JSONSchema
const GetResponseSchema = `
{
	"$id": "#SessionGetResponse",
	"type": "object",
	"properties": {
		"result": {
			"type": "object",
			"properties": {
				"session": { "$ref": "#Session" }
			}
		}
	}
}
`

/*
	@Operation POST /session/get - Get current user sessions
		Get the sessions with specified ID of current user.

		@Tag User
		@SecurityRequirement access_key
		@SecurityRequirement access_token

		@RequestBody
			Describe the session ID.
			@JSONSchema {SessionGetRequest}

		@Response 200
			The requested session.
			@JSONSchema {SessionGetResponse}
*/
type GetHandler struct {
	AuthContext     coreAuth.ContextGetter `dependency:"AuthContextGetter"`
	Validator       *validation.Validator  `dependency:"Validator"`
	RequireAuthz    handler.RequireAuthz   `dependency:"RequireAuthz"`
	TxContext       db.TxContext           `dependency:"TxContext"`
	SessionProvider session.Provider       `dependency:"SessionProvider"`
}

func (h GetHandler) ProvideAuthzPolicy() authz.Policy {
	return policy.AllOf(
		authz.PolicyFunc(policy.DenyNoAccessKey),
		policy.RequireValidUser,
	)
}

func (h GetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var response handler.APIResponse
	var payload GetRequestPayload
	if err := handler.BindJSONBody(r, w, h.Validator, "#SessionGetRequest", &payload); err != nil {
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

func (h GetHandler) Handle(payload GetRequestPayload) (resp interface{}, err error) {
	err = db.WithTx(h.TxContext, func() error {
		authInfo, _ := h.AuthContext.AuthInfo()
		userID := authInfo.ID
		sessionID := payload.SessionID

		s, err := h.SessionProvider.Get(sessionID)
		if err != nil {
			if errors.Is(err, session.ErrSessionNotFound) {
				err = errSessionNotFound
			}
			return err
		}
		if s.UserID != userID {
			return errSessionNotFound
		}

		resp = GetResponse{Session: authSession.Format(s)}
		return nil
	})
	return
}
