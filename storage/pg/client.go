package pg

import (
	"fmt"
	"time"

	"github.com/RangelReale/osin"
)

var _ = fmt.Sprintf
var _ osin.Client = (*Client)(nil)

type JsonKV map[string]interface{}

func toJsonKV(src interface{}) (JsonKV, error) {
	switch s := src.(type) {
	case JsonKV:
		return s, nil
	case map[string]interface{}:
		return JsonKV(s), nil
	}
	return nil, errInvalidJsonb
}

type Client struct {
	TableName struct{} `sql:"oauth.client" json:"-"`

	Code                 string    `sql:"code,pk" json:"code"`
	Name                 string    `sql:"name" json:"name"`
	Secret               string    `sql:"secret" json:"-"`
	RedirectUri          string    `sql:"redirect_uri" json:"redirect_uri"`
	UserData             JsonKV    `sql:"userdata" json:"userdata,omitempty"`
	CreatedAt            time.Time `sql:"created" json:"created,omitempty"`
	AllowedGrantTypes    []string  `sql:"allowed_grant_types" json:"grant_types,omitempty"`
	AllowedResponseTypes []string  `sql:"allowed_response_types" json:"response_types,omitempty"`
	AllowedScopes        []string  `sql:"allowed_scopes" json:"scopes,omitempty"`
}

// func (c *Client) String() string {
// 	return fmt.Sprintf("<oauth:Client code=%s>", c.Code)
// }

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
	return c.UserData
}

func NewClient(name, code, secret, redirectUri string) (c *Client) {
	c = &Client{
		Name:                 name,
		Code:                 code,
		Secret:               secret,
		RedirectUri:          redirectUri,
		CreatedAt:            time.Now(),
		AllowedGrantTypes:    []string{"authorization_code", "refresh_token"},
		AllowedResponseTypes: []string{},
		AllowedScopes:        []string{"basic"},
	}
	return
}
