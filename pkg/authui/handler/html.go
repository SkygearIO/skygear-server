package handler

import (
	"net/http"
	"strconv"
)

func WriteHTML(w http.ResponseWriter, body string) {
	b := []byte(body)
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(200)
	w.Write(b)
}
