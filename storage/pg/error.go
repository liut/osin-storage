package pg

import (
	"errors"
)

var (
	errNotFound    = errors.New("Not found.")
	errDatabase    = errors.New("Database error.")
	errNilClient   = errors.New("data.Client must not be nil")
	errInvalidJson = errors.New("Invalid JSON")
)
