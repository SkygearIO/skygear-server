package resolver

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/model"
	"github.com/skygeario/skygear-server/pkg/server/authtoken"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
)

type masterkeyAuthContextResolver struct {
	TokenStore    authtoken.Store `dependency:"TokenStore"`
	AuthInfoStore authinfo.Store  `dependency:"AuthInfoStore"`
}

func (r masterkeyAuthContextResolver) Resolve(req *http.Request) (ctx handler.AuthContext, err error) {
	tokenStr := model.GetAccessToken(req)
	token := &authtoken.Token{}
	r.TokenStore.Get(tokenStr, token)

	if token.AuthInfoID == "" {
		token.AuthInfoID = "_god"
	}

	info := &authinfo.AuthInfo{}
	if err = r.AuthInfoStore.GetAuth(token.AuthInfoID, info); err == skydb.ErrUserNotFound {
		info.ID = token.AuthInfoID

		if err = r.AuthInfoStore.CreateAuth(info); err == skydb.ErrUserDuplicated {
			// user already exists, error can be ignored
			err = nil
		}
	}

	ctx.AuthInfo = info

	return
}
