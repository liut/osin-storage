package oauth

import (
	"fmt"
	"log"
	"time"

	"github.com/liut/osin-storage/storage"
)

var _ storage.Client = (*Client)(nil)

// Client of oauth2
type Client struct {
	ID          string     `json:"id" db:"id" ` // pk
	Secret      string     `json:"secret" db:"secret"`
	RedirectURI string     `json:"redirectURI" db:"redirect_uri" `
	Meta        ClientMeta `json:"meta,omitempty" db:"meta" `       // jsonb
	CreatedAt   time.Time  `json:"created,omitempty" db:"created" ` // time.Now()
}

func (c *Client) String() string {
	return fmt.Sprintf("Client{ID: \"%s\" redirectURI: %q meta: %v}", c.ID, c.RedirectURI, c.Meta)
}

// GetId oauth.Client
func (c *Client) GetId() string {
	return c.ID
}

// GetSecret oauth.Client
func (c *Client) GetSecret() string {
	return c.Secret
}

// GetRedirectUri oauth.Client
func (c *Client) GetRedirectUri() string {
	return c.RedirectURI
}

// GetUserData oauth.Client
func (c *Client) GetUserData() interface{} {
	return c.Meta
}

// GetName ...
func (c *Client) GetName() string {
	return c.Meta.Name
}

// GetGrantTypes ...
func (c *Client) GetGrantTypes() []string {
	return c.Meta.GrantTypes
}

// GetResponseTypes ...
func (c *Client) GetResponseTypes() []string {
	return c.Meta.ResponseTypes
}

// GetScopes ...
func (c *Client) GetScopes() []string {
	return c.Meta.Scopes
}

// CopyFrom ...
func (c *Client) CopyFrom(other storage.Client) {
	c.ID = other.GetId()
	c.Secret = other.GetSecret()
	c.RedirectURI = other.GetRedirectUri()

	data := other.GetUserData()
	if extra, ok := data.(ClientMeta); ok {
		c.Meta = extra
	} else {
		log.Printf("invalid userData %v", data)
	}
}

// NewClient build a client
func NewClient(id, secret, redirectURI string) (c *Client) {
	c = &Client{
		ID:          id,
		Secret:      secret,
		RedirectURI: redirectURI,
		CreatedAt:   time.Now(),
		Meta:        defaultClientMeta,
	}
	return
}

// ClientSpec 查询参数
type ClientSpec struct {
	Page   int      `json:"page,omitempty" form:"page"`
	Limit  int      `json:"limit,omitempty" form:"limit"`
	Orders []string `json:"order,omitempty" form:"order"`
	Total  int      `json:"total,omitempty"` // for set value

	CountOnly bool `json:"count,omitempty" form:"count"`
}
