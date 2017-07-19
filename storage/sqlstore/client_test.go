package sqlstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	code        = "111"
	secret      = "secret"
	redirectUri = "http://localhost"
	meta        = defaultClientMeta
)

func TestClient(t *testing.T) {
	meta.Name = "test"
	c := NewClient(code, secret, redirectUri)
	c.Meta = meta

	assert.Equal(t, code, c.GetId(), c.Code)
	assert.Equal(t, secret, c.GetSecret(), c.Secret)
	assert.Equal(t, redirectUri, c.GetRedirectUri(), c.RedirectUri)
	assert.Equal(t, meta, c.Meta)
	assert.Equal(t, "test", c.GetName())

}

func TestJsonKV(t *testing.T) {
	m := JsonKV{"name": "eagle"}
	assert.Equal(t, m.WithKey("name"), "eagle")
}
