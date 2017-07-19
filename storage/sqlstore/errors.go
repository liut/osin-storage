package sqlstore

import (
	"errors"
)

var (
	dbError        = errors.New("database error")
	ErrNotFound    = errors.New("Not Found")
	valueError     = errors.New("value error")
	ErrInvalidJSON = errors.New("Invalid JSON")
)
