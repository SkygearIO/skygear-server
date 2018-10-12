package middleware

import (
	"net/http"

	"github.com/skygeario/skygear-server/pkg/server/uuid"
)

// RequestID add random request id to request context
type RequestID struct{}

func (m RequestID) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New()
		r.Header.Set("X-Skygear-Request-ID", requestID)
		w.Header().Set("X-Skygear-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}
