package webapp

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/core/db"
)

func AttachResetPasswordSuccessHandler(
	router *mux.Router,
	authDependency auth.DependencyMap,
) {
	router.
		NewRoute().
		Path("/reset_password/success").
		Methods("OPTIONS", "GET").
		Handler(auth.MakeHandler(authDependency, newResetPasswordSuccessHandler))
}

type resetPasswordSuccessProvider interface {
	GetResetPasswordSuccess(w http.ResponseWriter, r *http.Request) (func(error), error)
}

type ResetPasswordSuccessHandler struct {
	Provider  resetPasswordSuccessProvider
	TxContext db.TxContext
}

func (h *ResetPasswordSuccessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db.WithTx(h.TxContext, func() error {
		if r.Method == "GET" {
			writeResponse, err := h.Provider.GetResetPasswordSuccess(w, r)
			writeResponse(err)
			return err
		}
		return nil
	})
}
