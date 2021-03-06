package pg

import (
	"github.com/openshift/osin"
)

type Access struct {
	tableName struct{} `sql:"oauth.access" `

	Id int `sql:"id,pk" json:"id"`

	*osin.AccessData
}

var tables = []string{"oauth.client", "oauth.access", "oauth.refresh", "oauth.authorize"}
var schemas = []string{
	"CREATE SCHEMA IF NOT EXISTS oauth",
}
