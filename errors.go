package goblet

import (
	"errors"
)

var (
	// ErrKeyNotFound is Error of specified name not found
	ErrKeyNotFound = errors.New("ErrKeyNotFound")

	// ErrEmptyName is
	ErrEmptyName = errors.New("ErrEmptyName")

	// ErrCannotCall is
	ErrCannotCall = errors.New("ErrCannotCall")
)
