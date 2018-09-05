package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/gorilla/mux"

	"github.com/skygeario/skygear-server/pkg/gateway/middleware"
	"github.com/skygeario/skygear-server/pkg/gateway/db"
	"github.com/skygeario/skygear-server/pkg/gateway/db/pq"
)

var routerMap map[string]*url.URL

func init() {
	auth, _ := url.Parse("http://localhost:3000")
	routerMap = map[string]*url.URL{
		"auth": auth,
	}
}

func main() {
	ensureDB()

	r := mux.NewRouter()

	r.Use(middleware.TenantMiddleware{}.Handle)

	proxy := NewReverseProxy()
	r.HandleFunc("/{gear}/{rest:.*}", rewriteHandler(proxy))

	srv := &http.Server{
		Addr: "0.0.0.0:3001",
		// Good practice to set timeouts to avoid Slowloris attacks.
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      r, // Pass our instance of gorilla/mux in.
	}

	logger := log.New()

	logger.Info("Start gateway server")
	if err := srv.ListenAndServe(); err != nil {
		logger.Errorf("Fail to start gateway server %v", err)
	}
}

func NewReverseProxy() *httputil.ReverseProxy {
	director := func(req *http.Request) {
		path := req.URL.Path
		req.URL = routerMap[req.Header.Get("X-Skygear-Gear")]
		req.URL.Path = path
	}
	return &httputil.ReverseProxy{Director: director}
}

func rewriteHandler(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Skygear-Gear", mux.Vars(r)["gear"])
		r.URL.Path = "/" + mux.Vars(r)["rest"]
		p.ServeHTTP(w, r)
	}
}

func ensureDB() func() (db.Conn, error) {
	logger := log.New()
	// FIXME(carmenlau): Remove hard coded connection string
	connOpener := func() (db.Conn, error) {
		return pq.Open(
			context.Background(),
			"postgres://postgres:@localhost/postgres?sslmode=disable",
		)
	}

	// Attempt to open connection to database. Retry for a number of
	// times before giving up.
	attempt := 0
	for {
		conn, connError := connOpener()
		if connError == nil {
			conn.Close()
			return connOpener
		}

		attempt++
		logger.Errorf("Failed to start skygear: %v", connError)
		if attempt >= 5 {
			logger.Fatalf("Failed to start skygear server because connection to database cannot be opened.")
		}

		logger.Info("Retrying in 1 second...")
		time.Sleep(time.Second * time.Duration(1))
	}
}
