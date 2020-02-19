package provider

import (
	"net/http"

	coreAuth "github.com/skygeario/skygear-server/pkg/core/auth"
)

type AuthContextProvider interface {
	coreAuth.ContextGetter
	Init(r *http.Request)
}
