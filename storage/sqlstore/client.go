package sqlstore

import (
	"fmt"
	"log"
	"time"

	"github.com/liut/osin-storage/storage"
)

var _ storage.Client = (*Client)(nil)
var (
	defaultGrantTypes    = []string{"authorization_code", "password", "refresh_token"}
	defaultResponseTypes = []string{}
	defaultScopes        = []string{"basic"}
	defaultClientMeta    = ClientMeta{
		Name:          "",
		GrantTypes:    defaultGrantTypes,
		ResponseTypes: defaultResponseTypes,
		Scopes:        defaultScopes,
	}
)

type Client struct {
	Id          int        `sql:"id" json:"_id,omitempty"`
	Code        string     `sql:"code" json:"code"` // unique
	Secret      string     `sql:"secret" json:"-"`
	RedirectUri string     `sql:"redirect_uri" json:"redirect_uri"`
	Meta        ClientMeta `sql:"meta" json:"meta,omitempty"` // UserMeta
	CreatedAt   time.Time  `sql:"created" json:"created,omitempty"`
}

func (c *Client) String() string {
	return fmt.Sprintf("<oauth:Client id=\"%d\" code=%q redirect_uri=%q />", c.Id, c.Code, c.RedirectUri)
}

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
		Meta:        defaultClientMeta,
	}
	return
}
