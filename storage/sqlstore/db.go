package sqlstore

import (
	"database/sql"
	"log"
	"os"
)

type Queryer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type DBer interface {
	Queryer
	Begin() (*sql.Tx, error)
}

type DBTxer interface {
	Queryer
	Rollback() error
	Commit() error
}

func (s *DbStorage) withTxQuery(query func(tx DBTxer) error) error {

	tx, err := s.db.Begin()
	if err == nil {
		if err = query(tx); err == nil {
			return tx.Commit()
		}
	}
	tx.Rollback()
	log.Printf("tx query ERR: %s", err)
	return err
}

func envOr(key, dft string) string {
	v := os.Getenv(key)
	if v == "" {
		return dft
	}
	return v
}
