package db

import (
	"context"
)

// Driver opens an connection to the underlying database.
type Driver interface {
	Open(ctx context.Context, optionString string) (Conn, error)
}

// The DriverFunc type is an adapter such that an ordinary function
// can be used as a Driver.
type DriverFunc func(ctx context.Context, optionString string) (Conn, error)

// Open returns a Conn by calling the DriverFunc itself.
func (f DriverFunc) Open(ctx context.Context, name string) (Conn, error) {
	return f(ctx, name)
}
