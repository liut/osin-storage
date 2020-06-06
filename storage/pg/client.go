package pg

import (
	"fmt"
	"log"
	"time"

	"github.com/liut/osin-storage/storage"
	"github.com/openshift/osin"
)

var _ = fmt.Sprintf
var _ storage.Client = (*Client)(nil)
var _ osin.Client = (*Client)(nil)

// JSONKV ..
type JSONKV map[string]interface{}

// ToJSONKV ...
func ToJSONKV(src interface{}) (JSONKV, error) {
	switch s := src.(type) {
	case JSONKV:
		return s, nil
	case map[string]interface{}:
		return JSONKV(s), nil
	}
	return nil, errInvalidJson
}

// WithKey ...
func (m JSONKV) WithKey(key string) (v interface{}) {
	var ok bool
	if v, ok = m[key]; ok {
		return
	}
	return
}

// ClientMeta ...
type ClientMeta struct {
	Site uint8  `json:"siteID"`
	Name string `json:"name"`
}

// Client ...
type Client struct {
	tableName struct{} `sql:"oauth.client"`

	ID          int        `sql:"id,pk" json:"id"`
	Code        string     `sql:"code,unique" json:"code"`
	Secret      string     `sql:"secret,notnull" json:"-"`
	RedirectURI string     `sql:"redirect_uri" json:"redirect_uri"`
	Meta        ClientMeta `sql:"meta" json:"meta,omitempty"`
	CreatedAt   time.Time  `sql:"created" json:"created,omitempty"`
}

// func (c *Client) String() string {
// 	return fmt.Sprintf("<oauth:Client code=%s>", c.Code)
// }

// GetName ...
func (c *Client) GetName() string {
	return c.Meta.Name
}

// GetId osin.Client
func (c *Client) GetId() string { // justifying
	return c.Code
}

// GetSecret osin.Client
func (c *Client) GetSecret() string {
	return c.Secret
}

// GetRedirectUri osin.Client
func (c *Client) GetRedirectUri() string {
	return c.RedirectURI
}

// GetUserData osin.Client
func (c *Client) GetUserData() interface{} {
	return c.Meta
}

// CopyFrom ...
func (c *Client) CopyFrom(other storage.Client) {
	c.Code = other.GetId()
	c.Secret = other.GetSecret()
	c.RedirectURI = other.GetRedirectUri()

	data := other.GetUserData()
	if extra, ok := data.(ClientMeta); ok {
		c.Meta = extra
	} else {
		log.Printf("invalid userData %v", data)
	}
}

// NewClient ...
func NewClient(code, secret, uri string) (c *Client) {
	c = &Client{
		Code:        code,
		Secret:      secret,
		RedirectURI: uri,
		CreatedAt:   time.Now(),
		Meta:        ClientMeta{Name: ""},
	}
	return
}
