package webapp

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/auth/dependency/auth"
)

type AuthEntryPointMiddleware struct{}

func (m AuthEntryPointMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := auth.GetUser(r.Context())
		requireLoginPrompt := r.URL.Query().Get("prompt") == "login"
		if user != nil && !requireLoginPrompt {
			RedirectToRedirectURI(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
