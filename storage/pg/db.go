package pg

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
)

var (
	ormScan     = orm.Scan
	dbErrNoRows = pg.ErrNoRows
)

// DB ...
type DB = pg.DB

// Tx ...
type Tx = pg.Tx

// Query ...
type Query = orm.Query

// ormDB of go-pg/pg/orm.DB
type ormDB interface {
	Model(model ...interface{}) *Query
	Select(model interface{}) error
	Insert(model ...interface{}) error
	Update(model interface{}) error
	Delete(model interface{}) error

	Exec(query interface{}, params ...interface{}) (Result, error)
	ExecOne(query interface{}, params ...interface{}) (Result, error)
	Query(model, query interface{}, params ...interface{}) (Result, error)
	QueryOne(model, query interface{}, params ...interface{}) (Result, error)
}

// Result ...
type Result interface {

	// RowsAffected returns the number of rows affected by SELECT, INSERT, UPDATE,
	// or DELETE queries. It returns -1 if query can't possibly affect any rows,
	// e.g. in case of CREATE or SHOW queries.
	RowsAffected() int

	// RowsReturned returns the number of rows returned by the query.
	RowsReturned() int
}
