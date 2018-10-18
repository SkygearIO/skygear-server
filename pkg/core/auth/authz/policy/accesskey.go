package policy

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/core/auth/authz"

	"github.com/skygeario/skygear-server/pkg/core/auth"
	"github.com/skygeario/skygear-server/pkg/core/model"
	"github.com/skygeario/skygear-server/pkg/server/skyerr"
)

func DenyNoAccessKey(r *http.Request, ctx auth.ContextGetter) error {
	keyType := ctx.AccessKeyType()
	if keyType == model.NoAccessKey {
		return skyerr.NewError(skyerr.AccessKeyNotAccepted, "api key required")
	}

	return nil
}

func RequireMasterKey(r *http.Request, ctx auth.ContextGetter) error {
	keyType := ctx.AccessKeyType()
	if keyType != model.MasterAccessKey {
		return skyerr.NewError(skyerr.AccessKeyNotAccepted, "master key required")
	}

	return nil
}

// this ensures that our structure conform to certain interfaces.
var (
	_ authz.PolicyFunc = DenyNoAccessKey
	_ authz.PolicyFunc = RequireMasterKey
)
