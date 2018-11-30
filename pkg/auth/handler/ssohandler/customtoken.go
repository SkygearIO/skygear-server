package ssohandler

import (
	"encoding/json"
	"net/http"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/provider/customtoken"
	"github.com/skygeario/skygear-server/pkg/auth/dependency/userprofile"
	"github.com/skygeario/skygear-server/pkg/auth/response"
	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/auth/authtoken"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz"
	"github.com/skygeario/skygear-server/pkg/core/auth/authz/policy"
	"github.com/skygeario/skygear-server/pkg/core/auth/role"
	"github.com/skygeario/skygear-server/pkg/core/db"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/inject"
	"github.com/skygeario/skygear-server/pkg/core/server"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
	"github.com/skygeario/skygear-server/pkg/server/skyerr"
)

// AttachCustomTokenLoginHandler attaches CustomTokenLoginHandler to server
func AttachCustomTokenLoginHandler(
	server *server.Server,
	authDependency auth.DependencyMap,
) *server.Server {
	server.Handle("/sso/custom_token/login", &CustomTokenLoginHandlerFactory{
		authDependency,
	}).Methods("OPTIONS", "POST")
	return server
}

// CustomTokenLoginHandlerFactory creates CustomTokenLoginHandler
type CustomTokenLoginHandlerFactory struct {
	Dependency auth.DependencyMap
}

// NewHandler creates new CustomTokenLoginHandler
func (f CustomTokenLoginHandlerFactory) NewHandler(request *http.Request) http.Handler {
	h := &CustomTokenLoginHandler{}
	inject.DefaultInject(h, f.Dependency, request)
	return handler.APIHandlerToHandler(h, h.TxContext)
}

// ProvideAuthzPolicy provides authorization policy of handler
func (f CustomTokenLoginHandlerFactory) ProvideAuthzPolicy() authz.Policy {
	return authz.PolicyFunc(policy.DenyNoAccessKey)
}

type customTokenLoginPayload struct {
	TokenString string                           `json:"token"`
	Claims      customtoken.SSOCustomTokenClaims `json:"-"`
}

func (payload customTokenLoginPayload) Validate() error {
	if err := payload.Claims.Validate(); err != nil {
		return skyerr.NewError(
			skyerr.InvalidCredentials,
			err.Error(),
		)
	}
	return nil
}

// CustomTokenLoginHandler handles custom login request
type CustomTokenLoginHandler struct {
	TxContext               db.TxContext         `dependency:"TxContext"`
	UserProfileStore        userprofile.Store    `dependency:"UserProfileStore"`
	RoleStore               role.Store           `dependency:"RoleStore"`
	TokenStore              authtoken.Store      `dependency:"TokenStore"`
	AuthInfoStore           authinfo.Store       `dependency:"AuthInfoStore"`
	CustomTokenAuthProvider customtoken.Provider `dependency:"CustomTokenAuthProvider"`
}

func (h CustomTokenLoginHandler) WithTx() bool {
	return true
}

// DecodeRequest decode request payload
func (h CustomTokenLoginHandler) DecodeRequest(request *http.Request) (handler.RequestPayload, error) {
	payload := customTokenLoginPayload{}
	var err error
	if err = json.NewDecoder(request.Body).Decode(&payload); err != nil {
		return nil, err
	}

	payload.Claims, err = h.CustomTokenAuthProvider.Decode(payload.TokenString)
	if err != nil {
		return nil, skyerr.NewError(skyerr.BadRequest, err.Error())
	}
	return payload, err
}

// Handle function handle custom token login
func (h CustomTokenLoginHandler) Handle(req interface{}) (resp interface{}, err error) {
	payload := req.(customTokenLoginPayload)
	var info authinfo.AuthInfo
	var userProfile userprofile.UserProfile

	h.handleLogin(payload, &info, &userProfile)

	// TODO: check disable

	// Create auth token
	tkn, err := h.TokenStore.NewToken(info.ID)
	if err != nil {
		panic(err)
	}

	if err = h.TokenStore.Put(&tkn); err != nil {
		panic(err)
	}

	resp = response.NewAuthResponse(info, userProfile, tkn.AccessToken)

	// Populate the activity time to user
	now := timeNow()
	info.LastSeenAt = &now
	if err = h.AuthInfoStore.UpdateAuth(&info); err != nil {
		err = skyerr.MakeError(err)
		return
	}

	// TODO: audit trail

	// TODO: welcome email

	return
}

func (h CustomTokenLoginHandler) handleLogin(payload customTokenLoginPayload, info *authinfo.AuthInfo, userProfile *userprofile.UserProfile) (err error) {
	createNewUser := false
	principal, err := h.CustomTokenAuthProvider.GetPrincipalByTokenPrincipalID(payload.Claims.Subject)
	if err != nil {
		if err != skydb.ErrUserNotFound {
			return
		}

		err = nil
		createNewUser = true
	}

	if createNewUser {
		now := timeNow()
		*info = authinfo.NewAuthInfo()
		info.LastLoginAt = &now

		// Get default roles
		defaultRoles, e := h.RoleStore.GetDefaultRoles()
		if e != nil {
			err = skyerr.NewError(skyerr.InternalQueryInvalid, "unable to query default roles")
			return
		}

		// Assign default roles
		info.Roles = defaultRoles

		// Create AuthInfo
		if e = h.AuthInfoStore.CreateAuth(info); e != nil {
			if e == skydb.ErrUserDuplicated {
				err = skyerr.NewError(skyerr.Duplicated, "user duplicated")
				return
			}

			// TODO:
			// return proper error
			err = skyerr.NewError(skyerr.UnexpectedError, "Unable to save auth info")
			return
		}

		principal := customtoken.NewPrincipal()
		principal.TokenPrincipalID = payload.Claims.Subject
		principal.UserID = info.ID
		err = h.CustomTokenAuthProvider.CreatePrincipal(principal)
	} else {
		if e := h.AuthInfoStore.GetAuth(principal.UserID, info); e != nil {
			if err == skydb.ErrUserNotFound {
				err = skyerr.NewError(skyerr.ResourceNotFound, "User not found")
				return
			}
			err = skyerr.NewError(skyerr.ResourceNotFound, "User not found")
			return
		}
	}

	// Create Profile
	userProfileFunc := func(userID string, authInfo *authinfo.AuthInfo, data userprofile.Data) (userprofile.UserProfile, error) {
		if createNewUser {
			return h.UserProfileStore.CreateUserProfile(userID, authInfo, data)
		}

		return h.UserProfileStore.UpdateUserProfile(userID, authInfo, data)
	}

	if *userProfile, err = userProfileFunc(info.ID, info, payload.Claims.RawProfile); err != nil {
		// TODO:
		// return proper error
		err = skyerr.NewError(skyerr.UnexpectedError, "Unable to save user profile")
		return
	}

	return
}
