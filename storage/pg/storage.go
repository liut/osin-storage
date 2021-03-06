// Package pg is a osin storage implementation for postgres.
package pg

import (
	"fmt"
	"log"
	"time"

	"github.com/openshift/osin"

	"github.com/liut/osin-storage/storage"
)

var _ storage.Storage = (*dbStore)(nil)

// Storage ...
type Storage interface {
	storage.Storage
	AllClients() ([]Client, error)
	CreateSchemas() error
}

// Storage implements interface "github.com/openshift/osin".Storage and interface "github.com/ory-am/osin-storage".Storage
type dbStore struct {
	db *DB
}

// New returns a new postgres storage instance.
func New(db *DB) Storage {
	return &dbStore{db}
}

// CreateSchemas creates the schemata, if they do not exist yet in the database. Returns an error if something went wrong.
func (s *dbStore) CreateSchemas() error {
	for k, schema := range schemas {
		if _, err := s.db.Exec(schema); err != nil {
			log.Printf("Error creating schema %d: %s", k, schema)
			return err
		}
	}
	return nil
}

// Clone the storage if needed. For example, using mgo, you can clone the session with session.Clone
// to avoid concurrent access problems.
// This is to avoid cloning the connection at each method access.
// Can return itself if not a problem.
func (s *dbStore) Clone() osin.Storage {
	return s
}

// Close the resources the Storage potentially holds (using Clone for example)
func (s *dbStore) Close() {
}

// GetClient loads the client by id
func (s *dbStore) GetClient(id string) (osin.Client, error) {
	var c = new(Client)
	err := s.db.Model(c).Where("id = ?", id).Select()
	if err == dbErrNoRows {
		return nil, errNotFound
	} else if err != nil {
		log.Printf("get client %s err: %s", id, err)
	}
	return c, err
}

// SaveClient stores the client in the database and returns an error, if something went wrong.
func (s *dbStore) SaveClient(c storage.Client) (err error) {
	_c := NewClient(c.GetId(), c.GetSecret(), c.GetRedirectUri())
	if _c.GetId() == "" {
		return errNilClient
	}
	data := c.GetUserData()
	if extra, ok := data.(ClientMeta); ok {
		_c.Meta = extra
	}
	err = s.db.RunInTransaction(func(tx *Tx) (err error) {
		var created time.Time
		_, err = tx.QueryOne(ormScan(&created), "SELECT created FROM oauth.client WHERE id = ?", _c.ID)
		if err == nil {
			_, err = tx.Model(_c).
				Column("secret", "redirect_uri", "meta").
				Where("id = ?", c.GetId()).
				Returning("*").
				Update()
		} else {
			err = tx.Insert(_c)
		}
		return
	})

	return
}

// RemoveClient removes a client (identified by id) from the database. Returns an error if something went wrong.
func (s *dbStore) RemoveClient(code string) (err error) {
	var c Client
	_, err = s.db.Model(&c).Where("id = ?", code).Delete()
	return
}

// SaveAuthorize saves authorize data.
func (s *dbStore) SaveAuthorize(data *osin.AuthorizeData) (err error) {
	if _, err = ToJSONKV(data.UserData); err != nil {
		log.Printf("authorized.userdata %+v", data.UserData)
		return
	}
	if data.UserData == nil {
		data.UserData = JSONKV{}
	}

	_, err = s.db.Exec(
		"INSERT INTO oauth.authorize (client_id, code, expires_in, scopes, redirect_uri, state, created, extra) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		data.Client.GetId(),
		data.Code,
		data.ExpiresIn,
		data.Scope,
		data.RedirectUri,
		data.State,
		data.CreatedAt,
		data.UserData,
	)
	if err != nil {
		log.Printf("SaveAuthorize error %s", err)
	}
	return
}

// LoadAuthorize looks up AuthorizeData by a code.
// Client information MUST be loaded together.
// Optionally can return error if expired.
func (s *dbStore) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	var data osin.AuthorizeData
	var extra JSONKV
	var cid string
	scan := ormScan(&cid, &data.Code, &data.ExpiresIn, &data.Scope, &data.RedirectUri, &data.State, &data.CreatedAt, &extra)
	_, err := s.db.QueryOne(scan, "SELECT client_id, code, expires_in, scopes, redirect_uri, state, created, extra FROM oauth.authorize WHERE code=? LIMIT 1", code)
	if err == dbErrNoRows {
		return nil, errNotFound
	} else if err != nil {
		log.Printf("db error: %s", err)
		return nil, errDatabase
	}
	data.UserData = extra

	c, err := s.GetClient(cid)
	if err != nil {
		return nil, err
	}

	if data.ExpireAt().Before(time.Now()) {
		return nil, fmt.Errorf("Token expired at %s.", data.ExpireAt().String())
	}

	data.Client = c
	return &data, nil
}

// RemoveAuthorize revokes or deletes the authorization code.
func (s *dbStore) RemoveAuthorize(code string) (err error) {
	_, err = s.db.Exec("DELETE FROM oauth.authorize WHERE code=?", code)
	return nil
}

// SaveAccess writes AccessData.
// If RefreshToken is not blank, it must save in a way that can be loaded using LoadRefresh.
func (s *dbStore) SaveAccess(data *osin.AccessData) (err error) {
	_, err = s.LoadAccess(data.AccessToken)
	if err == nil {
		return nil
	} else if err != errNotFound {
		return
	}
	prev := ""
	authorizeData := &osin.AuthorizeData{}

	if data.AccessData != nil {
		prev = data.AccessData.AccessToken
	}

	if data.AuthorizeData != nil {
		authorizeData = data.AuthorizeData
	}

	var (
		extra JSONKV
	)
	if extra, err = ToJSONKV(data.UserData); err != nil {
		log.Printf("access.userdata %+v", data.UserData)
		return
	}

	return s.db.RunInTransaction(func(tx *Tx) (err error) {
		if data.RefreshToken != "" {
			if err = s.saveRefresh(tx, data.RefreshToken, data.AccessToken); err != nil {
				log.Printf("save refresh error %s", err)
				return
			}
		}

		if data.Client == nil {
			log.Print("access.client is nil")
			return errNilClient
		}

		_, err = tx.Exec("INSERT INTO oauth.access (client_id, authorize_code, previous, access_token, refresh_token, expires_in, scopes, redirect_uri, created, extra) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			data.Client.GetId(), authorizeData.Code, prev, data.AccessToken, data.RefreshToken, data.ExpiresIn, data.Scope, data.RedirectUri, data.CreatedAt, extra)
		if err != nil {
			log.Printf("insert error %s", err)
			return err
		}
		log.Print("save access OK")

		return
	})

}

// LoadAccess retrieves access data by token. Client information MUST be loaded together.
// AuthorizeData and AccessData DON'T NEED to be loaded if not easily available.
// Optionally can return error if expired.
func (s *dbStore) LoadAccess(code string) (*osin.AccessData, error) {
	var cid, prevAccessToken, authorizeCode string
	var result osin.AccessData
	var extra JSONKV

	sc := ormScan(
		&cid,
		&authorizeCode,
		&prevAccessToken,
		&result.AccessToken,
		&result.RefreshToken,
		&result.ExpiresIn,
		&result.Scope,
		&result.RedirectUri,
		&result.CreatedAt,
		&extra,
	)
	_, err := s.db.QueryOne(sc,
		"SELECT client_id, authorize_code, previous, access_token, refresh_token, expires_in, scopes, redirect_uri, created, extra FROM oauth.access WHERE access_token=? LIMIT 1",
		code,
	)
	if err == dbErrNoRows {
		return nil, errNotFound
	} else if err != nil {
		return nil, errDatabase
	}

	result.UserData = extra
	client, err := s.GetClient(cid)
	if err != nil {
		return nil, err
	}

	result.Client = client
	result.AuthorizeData, _ = s.LoadAuthorize(authorizeCode)
	prevAccess, _ := s.LoadAccess(prevAccessToken)
	result.AccessData = prevAccess
	return &result, nil
}

// RemoveAccess revokes or deletes an AccessData.
func (s *dbStore) RemoveAccess(code string) (err error) {
	_, err = s.db.Exec("DELETE FROM oauth.access WHERE access_token=?", code)
	return
}

// LoadRefresh retrieves refresh AccessData. Client information MUST be loaded together.
// AuthorizeData and AccessData DON'T NEED to be loaded if not easily available.
// Optionally can return error if expired.
func (s *dbStore) LoadRefresh(code string) (*osin.AccessData, error) {
	var access string
	_, err := s.db.QueryOne(ormScan(&access), "SELECT access FROM oauth.refresh WHERE token=? LIMIT 1", code)
	if err == dbErrNoRows {
		return nil, errNotFound
	} else if err != nil {
		return nil, err
	}
	return s.LoadAccess(access)
}

// RemoveRefresh revokes or deletes refresh AccessData.
func (s *dbStore) RemoveRefresh(code string) error {
	_, err := s.db.Exec("DELETE FROM oauth.refresh WHERE token=?", code)
	return err
}

func (s *dbStore) saveRefresh(tx *Tx, refresh, access string) (err error) {
	_, err = tx.Exec("INSERT INTO oauth.refresh (token, access) VALUES (?, ?)", refresh, access)
	return
}

func (s *dbStore) AllClients() (data []Client, err error) {
	err = s.db.Model(&data).Select()
	return
}
