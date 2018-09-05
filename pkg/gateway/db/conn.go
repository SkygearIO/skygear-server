package db

type Conn interface {
	Close() error
}
