package handler

import (
	"net/http"

	"github.com/gorilla/mux"
)

type RootHandler struct{}

func NewRootHandler() *RootHandler {
	return &RootHandler{}
}

func AttachRootHandler(router *mux.Router) {
	router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		InjectRootHandler(r).ServeHTTP(w, r)
	})
}

func (h *RootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/authorize", http.StatusFound)
}
