package sqlstore

import (
	"database/sql"
	"log"
	"time"

	"github.com/RangelReale/osin"

	"github.com/liut/osin-storage/storage"
)

var (
	_ Storage = (*DbStorage)(nil)
)

type Storage interface {
	storage.Storage
	AllClients() ([]*Client, error)
	GetClientWithCode(code string) (*Client, error)
	LoadScopes() (scopes []*Scope, err error)
	IsAuthorized(client_id, username string) bool
	SaveAuthorized(client_id, username string) error
}

type DbStorage struct {
	db DBer
}

// New returns a new sql storage instance.
func New(db DBer) Storage {
	s := &DbStorage{db: db}

	return s
}

func (s *DbStorage) Clone() osin.Storage {
	return New(s.db)
}

func (s *DbStorage) Close() {

}

func (s *DbStorage) GetClient(id string) (c osin.Client, err error) {
	c, err = s.GetClientWithCode(id)
	// debug("GetClient: '%s'", id)
	if err != nil {
		log.Printf("Client %q not found", id)
	}
	return
}

func (s *DbStorage) SaveAuthorize(data *osin.AuthorizeData) error {
	if _, err := ToJsonKV(data.UserData); err != nil {
		log.Printf("SaveAuthorize userdata %+v, ERR %s", data.UserData, err)
		return err
	}
	if data.UserData == nil {
		data.UserData = JsonKV{}
	}

	// debug("SaveAuthorize: '%s', extra '%s'\n", data.Code, extra)
	r, err := s.db.Exec(`INSERT INTO oauth.authorize(code, client_id, extra, redirect_uri, expires_in, scopes, created)
		    VALUES($1, $2, $3, $4, $5, $6, $7);`,
		data.Code, data.Client.GetId(), data.UserData,
		data.RedirectUri, data.ExpiresIn, data.Scope, data.CreatedAt)
	if err != nil {
		debug("SaveAuthorize: code '%s', extra '%s', result %v, ERR: %s", data.Code, data.UserData, r, err)
	}

	return err
}

func (s *DbStorage) LoadAuthorize(code string) (a *osin.AuthorizeData, err error) {
	// debug("LoadAuthorize: '%s'\n", code)
	var (
		client_id string
		extra     JsonKV
	)
	a = &osin.AuthorizeData{Code: code}
	err = s.db.QueryRow(`SELECT client_id, extra, redirect_uri, expires_in, scopes, created
		 FROM oauth.authorize WHERE code = $1`,
		code).Scan(&client_id, &extra, &a.RedirectUri, &a.ExpiresIn, &a.Scope, &a.CreatedAt)

	if err == nil {
		a.UserData = extra
		a.Client, err = s.GetClientWithCode(client_id)

		debug("loaded authorization '%s' ok, createdAt %s", code, a.CreatedAt)
		return
	}
	if err == sql.ErrNoRows {
		err = ErrNotFound
		return
	}
	debug("load authorize '%s' ERR: %s", code, err)
	log.Printf("Authorize %q not found", code)
	return
}

func (s *DbStorage) RemoveAuthorize(code string) error {
	// debug("RemoveAuthorize: '%s'\n", code)
	if code == "" {
		log.Print("authorize code is empty")
		return nil
	}
	qs := func(tx DBTxer) error {
		sql := `DELETE FROM oauth.authorize WHERE code = $1;`
		r, err := tx.Exec(sql, code)
		if err != nil {
			return err
		}

		debug("delete authorizeData code %s OK %v", code, r)

		return nil
	}
	return s.withTxQuery(qs)
}

func (s *DbStorage) SaveAccess(data *osin.AccessData) (err error) {
	// debug("SaveAccess: '%s'\n", data.AccessToken)
	_, err = s.LoadAccess(data.AccessToken)
	if err == nil {
		return nil
	} else if err != ErrNotFound {
		log.Printf("load access '%s' ERR %s", data.AccessToken, err)
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
	qs := func(tx DBTxer) error {
		r, err := tx.Exec(`INSERT INTO oauth.access (client_id, authorize_code, previous, access_token, refresh_token, expires_in, scopes, redirect_uri, created, extra)
			    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			data.Client.GetId(), authorizeData.Code, prev, data.AccessToken, data.RefreshToken,
			data.ExpiresIn, data.Scope, data.RedirectUri, data.CreatedAt, extra)
		if err != nil {
			return err
		}

		debug("save AccessData token %s OK %v", data.AccessToken, r)

		if data.RefreshToken != "" {
			if err = s.saveRefresh(tx, data.RefreshToken, data.AccessToken); err != nil {
				log.Printf("save refresh error %s", err)
				return err
			}
		}

		return nil
	}
	return s.withTxQuery(qs)
}

func (s *DbStorage) LoadAccess(code string) (a *osin.AccessData, err error) {
	// debug("LoadAccess: '%s'", code)
	var (
		cid, authorizeCode, prevAccessToken string
		extra                               JsonKV
		is_frozen                           bool
		id                                  int
	)
	a = &osin.AccessData{AccessToken: code}

	err = s.db.QueryRow(`SELECT id, client_id, authorize_code, previous, access_token, refresh_token, expires_in, scopes, redirect_uri, created, extra, is_frozen
		   FROM oauth.access WHERE access_token = $1`,
		code).Scan(&id, &cid, &authorizeCode, &prevAccessToken,
		&a.AccessToken, &a.RefreshToken, &a.ExpiresIn, &a.Scope,
		&a.RedirectUri, &a.CreatedAt, &extra, &is_frozen)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		debug("load access '%s' error: %s", code, err)
		log.Printf("AccessToken %q not found", code)
		return nil, dbError
	}

	a.UserData = extra
	a.Client, err = s.GetClient(cid)
	if err != nil {
		return
	}
	a.AuthorizeData, _ = s.LoadAuthorize(authorizeCode)
	prevAccess, _ := s.LoadAccess(prevAccessToken)
	a.AccessData = prevAccess
	debug("LoadAccess id #%d, code '%s' expires: \n\t%s created \n\t%s expire_at \n\t%s now \n\tis_expired %v",
		id, code, a.CreatedAt, a.ExpireAt(), time.Now(), a.IsExpired())
	return
}

func (s *DbStorage) RemoveAccess(code string) error {
	// debug("RemoveAccess: %s\n", code)
	qs := func(tx DBTxer) error {
		str := `DELETE FROM oauth.access WHERE access_token = $1;`
		r, err := tx.Exec(str, code)
		if err != nil {
			debug("RemoveAccess '%s', ERR %s", code, err)
			return err
		}

		debug("RemoveAccess '%s' OK %v", code, r)

		return nil
	}
	return s.withTxQuery(qs)
}

func (s *DbStorage) LoadRefresh(code string) (*osin.AccessData, error) {
	// debug("LoadRefresh: %s\n", code)
	var access string
	err := s.db.QueryRow(`SELECT access FROM oauth.refresh WHERE token=$1 LIMIT 1`, code).Scan(&access)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
		// return nil, fmt.Errorf("RefreshToken %q not found", code)
	} else if err != nil {
		debug("LoadRefresh '%s' ERR %s", code, err)
		return nil, err
	}
	return s.LoadAccess(access)
}

func (s *DbStorage) saveRefresh(tx DBTxer, refresh, access string) (err error) {
	_, err = s.db.Exec("INSERT INTO oauth.refresh (token, access) VALUES ($1, $2)", refresh, access)
	return
}

func (s *DbStorage) RemoveRefresh(code string) error {
	log.Printf("RemoveRefresh: %s\n", code)
	_, err := s.db.Exec("DELETE FROM oauth.refresh WHERE token=$1", code)
	return err
}

func (s *DbStorage) GetClientWithCode(code string) (c *Client, err error) {
	c = new(Client)
	err = s.db.QueryRow("SELECT id, code, secret, redirect_uri, meta, created FROM oauth.client WHERE code = $1",
		code).Scan(&c.Id, &c.Code, &c.Secret, &c.RedirectUri, &c.Meta, &c.CreatedAt)
	if err == sql.ErrNoRows {
		log.Printf("GetClientWithCode '%s', ERR %s", code, err)
		err = ErrNotFound
	} else if err != nil {
		log.Printf("GetClientWithCode '%s' ERROR: %s", code, err)
	}
	return
}

func (s *DbStorage) AllClients() (clients []*Client, err error) {

	str := `SELECT id, code, secret, redirect_uri, meta, created
	   FROM oauth.client `

	clients = make([]*Client, 0)

	rows, err := s.db.Query(str)
	if err != nil {
		log.Printf("db query error: %s for sql %s", err, str)
		return
	}
	defer rows.Close()
	for rows.Next() {
		c := new(Client)
		err = rows.Scan(&c.Id, &c.Code, &c.Secret, &c.RedirectUri, &c.Meta, &c.CreatedAt)
		if err != nil {
			log.Printf("rows scan error: %s", err)
			continue
		}
		clients = append(clients, c)
	}
	err = rows.Err()

	return
}

// CreateClient stores the client in the database and returns an error, if something went wrong.
func (s *DbStorage) SaveClient(client storage.Client) error {
	c := new(Client)
	c.CopyFrom(client)
	if c.Code == "" || c.Secret == "" || c.RedirectUri == "" {
		return valueError
	}
	// debug("SaveClient: code %s, uri %s", c.Code, c.RedirectUri)

	qs := func(tx DBTxer) (err error) {
		var id int
		err = tx.QueryRow("SELECT id FROM oauth.client WHERE code = $1", c.Code).Scan(&id)
		if err != nil && err != sql.ErrNoRows {
			log.Printf("query client %s ERR %s", c.Code, err)
			return
		}
		if id > 0 {
			str := `UPDATE oauth.client SET meta = $1, code = $2, secret = $3, redirect_uri = $4
			 WHERE id = $5`
			var r sql.Result
			r, err = tx.Exec(str, c.Meta, c.Code, c.Secret, c.RedirectUri, id)
			log.Printf("UPDATE client result: %v", r)
		} else {
			str := `INSERT INTO
		 oauth.client(meta, code, secret, redirect_uri)
		 VALUES($1, $2, $3, $4) RETURNING id;`
			err = tx.QueryRow(str,
				c.Meta,
				c.Code,
				c.Secret,
				c.RedirectUri).Scan(&id)
			if err == nil {
				c.Id = id
			}
			debug("save new client %s", c)
		}
		return err
	}
	return s.withTxQuery(qs)
}

// RemoveClient removes a client (identified by id) from the database. Returns an error if something went wrong.
func (s *DbStorage) RemoveClient(code string) (err error) {
	_, err = s.db.Exec("DELETE FROM oauth.client WHERE code = $1", code)
	return
}

func (s *DbStorage) LoadScopes() (scopes []*Scope, err error) {
	scopes = make([]*Scope, 0)

	rows, err := s.db.Query("SELECT name, label, description, is_default FROM oauth.scope")
	if err != nil {
		log.Printf("load scopes error: %s", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		s := new(Scope)
		err = rows.Scan(&s.Name, &s.Label, &s.Description, &s.IsDefault)
		if err != nil {
			log.Printf("rows scan error: %s", err)
		}
		scopes = append(scopes, s)
	}
	err = rows.Err()

	return
}

func (s *DbStorage) IsAuthorized(client_id, username string) bool {
	var (
		created time.Time
	)
	err := s.db.QueryRow("SELECT created FROM oauth.client_user_authorized WHERE client_id = $1 AND username = $2",
		client_id, username).Scan(&created)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Printf("load IsAuthorized(%s, %s) ERROR: %s", client_id, username, err)
		}
		return false
	}
	return true
}

func (s *DbStorage) SaveAuthorized(client_id, username string) (err error) {
	_, err = s.db.Exec("INSERT INTO oauth.client_user_authorized(client_id, username) VALUES($1, $2) ",
		client_id, username)
	return
}
