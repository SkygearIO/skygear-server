package pq

import (
	"fmt"
	"net"
	"context"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/skygeario/skygear-server/pkg/gateway/db"
	"github.com/skygeario/skygear-server/pkg/gateway/db/migration"
)

func isNetworkError(err error) bool {
	_, ok := err.(*net.OpError)
	return ok
}

// Open returns a new connection to postgresql implementation
func Open(ctx context.Context, connString string) (db.Conn, error) {
	db, err := getDB(connString)
	if err != nil {
		return nil, err
	}

	return &conn{
		db:                     db,
		context:                ctx,
	}, nil
}

type getDBReq struct {
	connString string
	done       chan getDBResp
}

type getDBResp struct {
	db  *sqlx.DB
	err error
}

var dbs = map[string]*sqlx.DB{}
var getDBChan = make(chan getDBReq)

func getDB(connString string) (*sqlx.DB, error) {
	ch := make(chan getDBResp)
	getDBChan <- getDBReq{connString, ch}
	resp := <-ch
	return resp.db, resp.err
}

// goroutine that initialize the database for use
func dbInitializer() {
	for {
		req := <-getDBChan
		db, ok := dbs[req.connString]
		if !ok {
			var err error
			db, err = sqlx.Open("postgres", req.connString)
			if err != nil {
				req.done <- getDBResp{nil, fmt.Errorf("failed to open connection: %s", err)}
				continue
			}

			db.SetMaxOpenConns(10)

			if err := mustInitDB(db); err != nil {
				db.Close()
				req.done <- getDBResp{nil, fmt.Errorf("failed to open connection: %s", err)}
				continue
			}

			dbs[req.connString] = db
		}

		req.done <- getDBResp{db, nil}
	}
}

// mustInitDB initialize database objects.
func mustInitDB(db *sqlx.DB) error {
	schema := "app_config"
	err := migration.EnsureLatest(db, schema, true)

	if err != nil {
		if isNetworkError(err) {
			return fmt.Errorf("gateway/db: unable to connect to database because of a network error = %v", err)
		} else {
			return fmt.Errorf("gateway/db: unable to migrate database because of an error = %v", err)
		}
	}
	return nil
}

func init() {
	go dbInitializer()
}
