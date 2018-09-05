
package pq

import (
	"context"
	"errors"
	"database/sql"

	log "github.com/sirupsen/logrus"
	"github.com/jmoiron/sqlx"
)

var ErrDatabaseTxDidBegin = errors.New("A transaction has already begun")
var ErrDatabaseTxDidNotBegin = errors.New("A transaction has not begun")
var ErrDatabaseTxDone = errors.New("Database's transaction has already committed or rolled back")

// ExtContext is an interface for both sqlx.DB and sqlx.Tx
type ExtContext interface {
	sqlx.ExtContext
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type conn struct {
	db                     *sqlx.DB // database wrapper
	tx                     *sqlx.Tx // transaction wrapper, nil when no transaction
	context                context.Context
}

// Db returns the current database wrapper, or a transaction wrapper when
// a transaction is in effect.
func (c *conn) Db() ExtContext {
	if c.tx != nil {
		return c.tx
	}
	return c.db
}

// Begin begins a transaction.
func (c *conn) Begin() error {
	logger := log.New()
	logger.Debugf("%p: Beginning transaction", c)
	if c.tx != nil {
		return ErrDatabaseTxDidBegin
	}

	tx, err := c.db.Beginx()
	if err != nil {
		logger.Debugf("%p: Unable to begin transaction %p: %v", c, err)
		return err
	}
	c.tx = tx
	logger.Debugf("%p: Done beginning transaction %p", c, c.tx)
	return nil
}

// Commit commits a transaction.
func (c *conn) Commit() error {
	logger := log.New()
	if c.tx == nil {
		return ErrDatabaseTxDidNotBegin
	}

	if err := c.tx.Commit(); err != nil {
		logger.Errorf("%p: Unable to commit transaction %p: %v", c, c.tx, err)
		return err
	}
	c.tx = nil
	logger.Debugf("%p: Committed transaction", c)
	return nil
}

// Rollback rollbacks a transaction.
func (c *conn) Rollback() error {
	logger := log.New()
	if c.tx == nil {
		return ErrDatabaseTxDidNotBegin
	}

	if err := c.tx.Rollback(); err != nil {
		logger.Errorf("%p: Unable to rollback transaction %p: %v", c, c.tx, err)
		return err
	}
	c.tx = nil
	logger.Debugf("%p: Rolled back transaction", c)
	return nil
}

func (c *conn) Close() error { return nil }