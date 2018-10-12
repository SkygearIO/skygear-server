package middleware

import (
	"net/http"

	nextSkyerr "github.com/skygeario/skygear-server/pkg/core/skyerr"
	"github.com/skygeario/skygear-server/pkg/server/skyerr"
)

// RecoverHandler provides an interface to handle recovered panic error
type RecoverHandler func(http.ResponseWriter, *http.Request, skyerr.Error)

// Recover recover from panic
type Recover struct {
	RecoverHandler RecoverHandler
}

func (m Recover) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); r != nil {
				err := nextSkyerr.ErrorFromRecoveringPanic(rec)
				if m.RecoverHandler != nil {
					m.RecoverHandler(w, r, err)
				}
			}
		}()

		next.ServeHTTP(w, r)
	})
}
