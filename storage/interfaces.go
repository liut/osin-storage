// Package storage defines an interface, which all osin-storage implementations are going to support.
package storage

import "github.com/openshift/osin"

// Client ...
type Client interface {
	osin.Client

	// GetName
	GetName() string

	// CopyFrom
	CopyFrom(Client)
}

// Storage extends github.com/openshift/osin.Storage with create, update and delete methods for clients.
type Storage interface {
	osin.Storage

	// CreateClient stores the client in the database and returns an error if something went wrong.
	SaveClient(client Client) error

	// RemoveClient removes a client (identified by id) from the database. Returns an error if something went wrong.
	RemoveClient(id string) error
}
