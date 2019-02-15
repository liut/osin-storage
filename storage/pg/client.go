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

type JsonKV map[string]interface{}

func ToJsonKV(src interface{}) (JsonKV, error) {
	switch s := src.(type) {
	case JsonKV:
		return s, nil
	case map[string]interface{}:
		return JsonKV(s), nil
	}
	return nil, errInvalidJson
}

func (m JsonKV) WithKey(key string) (v interface{}) {
	var ok bool
	if v, ok = m[key]; ok {
		return
	}
	return
}

type ClientMeta struct {
	Site uint8  `json:"site_id"`
	Name string `json:"name"`
}

type Client struct {
	tableName struct{} `sql:"oauth.client" json:"-"`

	Id          int        `sql:"id,pk" json:"id"`
	Code        string     `sql:"code,unique" json:"code"`
	Secret      string     `sql:"secret,notnull" json:"-"`
	RedirectUri string     `sql:"redirect_uri" json:"redirect_uri"`
	Meta        ClientMeta `sql:"meta" json:"meta,omitempty"`
	CreatedAt   time.Time  `sql:"created" json:"created,omitempty"`
}

// func (c *Client) String() string {
// 	return fmt.Sprintf("<oauth:Client code=%s>", c.Code)
// }

func (c *Client) GetName() string {
	return c.Meta.Name
}

func (c *Client) GetId() string {
	return c.Code
}

func (c *Client) GetSecret() string {
	return c.Secret
}

func (c *Client) GetRedirectUri() string {
	return c.RedirectUri
}

func (c *Client) GetUserData() interface{} {
	return c.Meta
}

func (c *Client) CopyFrom(other storage.Client) {
	c.Code = other.GetId()
	c.Secret = other.GetSecret()
	c.RedirectUri = other.GetRedirectUri()

	data := other.GetUserData()
	if extra, ok := data.(ClientMeta); ok {
		c.Meta = extra
	} else {
		log.Printf("invalid userData %v", data)
	}
}

func NewClient(code, secret, redirectUri string) (c *Client) {
	c = &Client{
		Code:        code,
		Secret:      secret,
		RedirectUri: redirectUri,
		CreatedAt:   time.Now(),
		Meta:        ClientMeta{Name: ""},
	}
	return
}
