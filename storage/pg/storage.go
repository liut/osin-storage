// Package postgres is a osin storage implementation for postgres.
package pg

import (
	"fmt"
	"log"
	"time"

	"github.com/RangelReale/osin"
	"gopkg.in/pg.v4"
	"gopkg.in/pg.v4/orm"
)

// Storage implements interface "github.com/RangelReale/osin".Storage and interface "github.com/ory-am/osin-storage".Storage
type Storage struct {
	db *pg.DB
}

// New returns a new postgres storage instance.
func New(db *pg.DB) *Storage {
	return &Storage{db}
}

// CreateSchemas creates the schemata, if they do not exist yet in the database. Returns an error if something went wrong.
func (s *Storage) CreateSchemas() error {
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
func (s *Storage) Clone() osin.Storage {
	return s
}

// Close the resources the Storage potentially holds (using Clone for example)
func (s *Storage) Close() {
}

// GetClient loads the client by id
func (s *Storage) GetClient(id string) (osin.Client, error) {
	var c = new(Client)
	err := orm.NewQuery(s.db, c).Where("code = ?", id).Select()
	if err == pg.ErrNoRows {
		return nil, errNotFound
	} else if err != nil {
		log.Printf("get client %s err: %s", id, err)
	}
	return c, err
}

// UpdateClient updates the client (identified by it's id) and replaces the values with the values of client.
func (s *Storage) UpdateClient(c osin.Client) (err error) {
	_c := NewClient(c.GetId(), c.GetSecret(), c.GetRedirectUri())
	data := c.GetUserData()
	if extra, ok := data.(ClientMeta); ok {
		_c.UserData = extra
	}
	_, err = orm.NewQuery(s.db, _c).
		Column("secret", "redirect_uri", "userdata").
		Where("code = ?", c.GetId()).
		Returning("*").
		Update()

	return
}

// CreateClient stores the client in the database and returns an error, if something went wrong.
func (s *Storage) CreateClient(c osin.Client) (err error) {
	_c := NewClient(c.GetId(), c.GetSecret(), c.GetRedirectUri())
	if _c.GetId() == "" {
		return errNilClient
	}
	data := c.GetUserData()
	if extra, ok := data.(ClientMeta); ok {
		_c.UserData = extra
	}

	err = orm.Create(s.db, _c)
	return
}

// RemoveClient removes a client (identified by id) from the database. Returns an error if something went wrong.
func (s *Storage) RemoveClient(code string) (err error) {
	var c Client
	_, err = orm.NewQuery(s.db, &c).Where("code = ?", code).Delete()
	return
}

// SaveAuthorize saves authorize data.
func (s *Storage) SaveAuthorize(data *osin.AuthorizeData) (err error) {
	// extra, err := assertToString(data.UserData)
	// if err != nil {
	// 	return err
	// }
	if _, err = ToJsonKV(data.UserData); err != nil {
		log.Printf("authorized.userdata %+v", data.UserData)
		return
	}
	if data.UserData == nil {
		data.UserData = JsonKV{}
	}

	_, err = s.db.Exec(
		"INSERT INTO oauth.authorize (client, code, expires_in, scopes, redirect_uri, state, created, extra) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
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
func (s *Storage) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	var data osin.AuthorizeData
	var extra JsonKV
	var cid string
	scan := orm.Scan(&cid, &data.Code, &data.ExpiresIn, &data.Scope, &data.RedirectUri, &data.State, &data.CreatedAt, &extra)
	_, err := s.db.QueryOne(scan, "SELECT client, code, expires_in, scopes, redirect_uri, state, created, extra FROM oauth.authorize WHERE code=? LIMIT 1", code)
	if err == pg.ErrNoRows {
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
func (s *Storage) RemoveAuthorize(code string) (err error) {
	_, err = s.db.Exec("DELETE FROM oauth.authorize WHERE code=?", code)
	return nil
}

// SaveAccess writes AccessData.
// If RefreshToken is not blank, it must save in a way that can be loaded using LoadRefresh.
func (s *Storage) SaveAccess(data *osin.AccessData) (err error) {
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
		extra JsonKV
	)
	if extra, err = ToJsonKV(data.UserData); err != nil {
		log.Printf("access.userdata %+v", data.UserData)
		return
	}

	return s.db.RunInTransaction(func(tx *pg.Tx) (err error) {
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

		_, err = tx.Exec("INSERT INTO oauth.access (client, authorize_code, previous, access_token, refresh_token, expires_in, scopes, redirect_uri, created, extra) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", data.Client.GetId(), authorizeData.Code, prev, data.AccessToken, data.RefreshToken, data.ExpiresIn, data.Scope, data.RedirectUri, data.CreatedAt, extra)
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
func (s *Storage) LoadAccess(code string) (*osin.AccessData, error) {
	var cid, prevAccessToken, authorizeCode string
	var result osin.AccessData
	var extra JsonKV

	sc := orm.Scan(
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
		"SELECT client, authorize_code, previous, access_token, refresh_token, expires_in, scopes, redirect_uri, created, extra FROM oauth.access WHERE access_token=? LIMIT 1",
		code,
	)
	if err == pg.ErrNoRows {
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
func (s *Storage) RemoveAccess(code string) (err error) {
	_, err = s.db.Exec("DELETE FROM oauth.access WHERE access_token=?", code)
	return
}

// LoadRefresh retrieves refresh AccessData. Client information MUST be loaded together.
// AuthorizeData and AccessData DON'T NEED to be loaded if not easily available.
// Optionally can return error if expired.
func (s *Storage) LoadRefresh(code string) (*osin.AccessData, error) {
	var access string
	_, err := s.db.QueryOne(orm.Scan(&access), "SELECT access FROM oauth.refresh WHERE token=? LIMIT 1", code)
	if err == pg.ErrNoRows {
		return nil, errNotFound
	} else if err != nil {
		return nil, err
	}
	return s.LoadAccess(access)
}

// RemoveRefresh revokes or deletes refresh AccessData.
func (s *Storage) RemoveRefresh(code string) error {
	_, err := s.db.Exec("DELETE FROM oauth.refresh WHERE token=?", code)
	return err
}

func (s *Storage) saveRefresh(tx txDber, refresh, access string) (err error) {
	_, err = tx.Exec("INSERT INTO oauth.refresh (token, access) VALUES (?, ?)", refresh, access)
	return
}

func assertToString(in interface{}) (string, error) {
	var ok bool
	var data string
	if in == nil {
		return "", nil
	} else if data, ok = in.(string); ok {
		return data, nil
	} else if str, ok := in.(fmt.Stringer); ok {
		return str.String(), nil
	}
	return "", fmt.Errorf(`Could not assert "%v" to string`, in)
}
