package webapp

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/skygeario/skygear-server/pkg/auth"
	"github.com/skygeario/skygear-server/pkg/core/db"
)

func AttachEnterPasswordHandler(
	router *mux.Router,
	authDependency auth.DependencyMap,
) {
	router.
		NewRoute().
		Path("/enter_password").
		Methods("OPTIONS", "POST", "GET").
		Handler(auth.MakeHandler(authDependency, newEnterPasswordHandler))
}

type enterPasswordProvider interface {
	GetEnterPasswordForm(w http.ResponseWriter, r *http.Request) (func(err error), error)
	EnterSecret(w http.ResponseWriter, r *http.Request) (func(err error), error)
}

type EnterPasswordHandler struct {
	Provider  enterPasswordProvider
	TxContext db.TxContext
}

func (h *EnterPasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	db.WithTx(h.TxContext, func() error {
		if r.Method == "GET" {
			writeResponse, err := h.Provider.GetEnterPasswordForm(w, r)
			writeResponse(err)
			return err
		}

		if r.Method == "POST" {
			writeResponse, err := h.Provider.EnterSecret(w, r)
			writeResponse(err)
			return err
		}
		return nil
	})
}
