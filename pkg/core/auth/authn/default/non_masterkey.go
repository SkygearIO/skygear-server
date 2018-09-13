package authn

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/core/auth/authinfo"
	"github.com/skygeario/skygear-server/pkg/core/handler"
	"github.com/skygeario/skygear-server/pkg/core/model"
	"github.com/skygeario/skygear-server/pkg/server/authtoken"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
)

type nonMasterkeyAuthContextResolver struct {
	TokenStore    authtoken.Store `dependency:"TokenStore"`
	AuthInfoStore authinfo.Store  `dependency:"AuthInfoStore"`
}

func (r nonMasterkeyAuthContextResolver) Resolve(req *http.Request) (ctx handler.AuthContext, err error) {
	tokenStr := model.GetAccessToken(req)

	token := &authtoken.Token{}
	err = r.TokenStore.Get(tokenStr, token)
	if err != nil {
		// TODO:
		// handle error properly
		return
	}

	ctx.Token = token

	info := &skydb.AuthInfo{}
	err = r.AuthInfoStore.GetAuth(token.AuthInfoID, info)
	if err != nil {
		// TODO:
		// handle error properly
		return
	}

	ctx.AuthInfo = info

	return
}
