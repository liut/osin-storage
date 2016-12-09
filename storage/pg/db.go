package pg

import (
	"errors"

	"gopkg.in/pg.v5/types"
)

var (
	errNotFound     = errors.New("Not found.")
	errDatabase     = errors.New("Database error.")
	errNilClient    = errors.New("data.Client must not be nil")
	errInvalidJsonb = errors.New("Invalid JSON")
)

type dber interface {
	Exec(q interface{}, params ...interface{}) (*types.Result, error)
	ExecOne(q interface{}, params ...interface{}) (*types.Result, error)
	Query(coll, query interface{}, params ...interface{}) (*types.Result, error)
	QueryOne(model, query interface{}, params ...interface{}) (*types.Result, error)
	FormatQuery(dst []byte, query string, params ...interface{}) []byte
}

type txDber interface {
	dber
	Rollback() error
	Commit() error
}
