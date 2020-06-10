package sqlstore

import (
	"github.com/liut/osin-storage/storage/oauth"
)

type Client = oauth.Client
type ClientMeta = oauth.ClientMeta
type JSONKV = oauth.JSONKV

func NewClient(id, secret, redirectURI string) *oauth.Client {
	return oauth.NewClient(id, secret, redirectURI)
}
