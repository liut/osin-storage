package pg

import (
	"fmt"
	"log"
	"os"
	// "reflect"
	"testing"
	"time"

	"github.com/RangelReale/osin"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/pg.v5"

	"github.com/liut/osin-storage/storage"
)

var _ = fmt.Sprintf
var db *pg.DB
var store Storage
var clientMetaEmpty = ClientMeta{}
var userDataEmpty = JsonKV{}
var userDataMock = JsonKV{"name": "foobar"}

func init() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	pg.SetLogger(log.New(os.Stderr, "pg>", log.Ltime|log.Lshortfile))
}

func TestMain(m *testing.M) {
	db = pg.Connect(&pg.Options{
		User:     "sso",
		Password: "sso",
		Addr:     "127.0.0.1:54320",
		Database: "sso",
	})

	store = New(db)
	if err := store.CreateSchemas(); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}

	retCode := m.Run()

	// force teardown
	tearDown()

	os.Exit(retCode)
}

func tearDown() {
	db.Exec("TRUNCATE TABLE oauth.access")
	db.Exec("TRUNCATE TABLE oauth.authorize")
	db.Exec("DELETE FROM oauth.client WHERE code in ('1', '3', 'dupe')")
	db.Exec("TRUNCATE TABLE oauth.refresh")
}

func TestClientOperations(t *testing.T) {
	create := &Client{Code: "1", Secret: "secret", RedirectUri: "http://localhost/", Meta: clientMetaEmpty}
	createClient(t, store, create)
	getClient(t, store, create)

	update := &Client{Code: "1", Secret: "secret123", RedirectUri: "http://www.google.com/", Meta: clientMetaEmpty}
	updateClient(t, store, update)
	getClient(t, store, update)

	data, err := store.AllClients()
	assert.Nil(t, err)
	assert.True(t, len(data) > 0)

}

func TestAuthorizeOperations(t *testing.T) {
	// client := &Client{Code: "2", Secret: "secret", RedirectUri: "http://localhost/", Meta: userDataEmpty}
	client := NewClient("2", "secret", "http://localhost/")
	client.Meta = clientMetaEmpty
	createClient(t, store, client)

	for _, authorize := range []*osin.AuthorizeData{
		{
			Client:      client,
			Code:        uuid.New(),
			ExpiresIn:   int32(600),
			Scope:       "scope",
			RedirectUri: "http://localhost/",
			State:       "state",
			CreatedAt:   time.Now().Round(time.Second),
			UserData:    userDataMock,
		},
	} {
		// Test save
		require.Nil(t, store.SaveAuthorize(authorize))

		// Test fetch
		result, err := store.LoadAuthorize(authorize.Code)
		require.Nil(t, err)
		require.Equal(t, authorize.CreatedAt.Unix(), authorize.CreatedAt.Unix())
		authorize.CreatedAt = result.CreatedAt
		//require.True(t, reflect.DeepEqual(authorize, result), "Case: %d\n%v\n\n%v", k, authorize, result)
		// require.EqualValues(t, authorize, result)
		require.Equal(t, authorize.Code, result.Code)
		require.Equal(t, authorize.ExpiresIn, result.ExpiresIn)
		require.Equal(t, authorize.CreatedAt, result.CreatedAt)
		require.Equal(t, authorize.Client.GetId(), result.Client.GetId())

		// Test remove
		require.Nil(t, store.RemoveAuthorize(authorize.Code))
		_, err = store.LoadAuthorize(authorize.Code)
		require.NotNil(t, err)
	}

	removeClient(t, store, client)
}

func TestStoreFailsOnInvalidUserData(t *testing.T) {
	// client := &Client{Code: "3", Secret: "secret", RedirectUri: "http://localhost/", Meta: userDataEmpty}
	client := NewClient("3", "secret", "http://localhost/")
	client.Meta = clientMetaEmpty
	authorize := &osin.AuthorizeData{
		Client:      client,
		Code:        uuid.New(),
		ExpiresIn:   int32(60),
		Scope:       "scope",
		RedirectUri: "http://localhost/",
		State:       "state",
		CreatedAt:   time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		UserData:    struct{ foo string }{"bar"},
	}
	access := &osin.AccessData{
		Client:        client,
		AuthorizeData: authorize,
		AccessData:    nil,
		AccessToken:   uuid.New(),
		RefreshToken:  uuid.New(),
		ExpiresIn:     int32(60),
		Scope:         "scope",
		RedirectUri:   "https://localhost/",
		CreatedAt:     time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
		UserData:      struct{ foo string }{"bar"},
	}
	assert.NotNil(t, store.SaveAuthorize(authorize))
	assert.NotNil(t, store.SaveAccess(access))
}

func TestAccessOperations(t *testing.T) {
	// client := &Client{Code: "3", Secret: "secret", RedirectUri: "http://localhost/", Meta: userDataEmpty}
	client := NewClient("3", "secret", "http://localhost/")
	authorize := &osin.AuthorizeData{
		Client:      client,
		Code:        uuid.New(),
		ExpiresIn:   int32(60),
		Scope:       "scope",
		RedirectUri: "http://localhost/",
		State:       "state",
		CreatedAt:   time.Now().Round(time.Second),
		UserData:    userDataEmpty,
	}
	nestedAccess := &osin.AccessData{
		Client:        client,
		AuthorizeData: authorize,
		AccessData:    nil,
		AccessToken:   uuid.New(),
		RefreshToken:  uuid.New(),
		ExpiresIn:     int32(60),
		Scope:         "scope",
		RedirectUri:   "https://localhost/",
		CreatedAt:     time.Now().Round(time.Second),
		UserData:      userDataMock,
	}
	access := &osin.AccessData{
		Client:        client,
		AuthorizeData: authorize,
		AccessData:    nestedAccess,
		AccessToken:   uuid.New(),
		RefreshToken:  uuid.New(),
		ExpiresIn:     int32(60),
		Scope:         "scope",
		RedirectUri:   "https://localhost/",
		CreatedAt:     time.Now().Round(time.Second),
		UserData:      userDataMock,
	}

	createClient(t, store, client)
	require.Nil(t, store.SaveAuthorize(authorize))
	require.Nil(t, store.SaveAccess(nestedAccess))
	require.Nil(t, store.SaveAccess(access))

	result, err := store.LoadAccess(access.AccessToken)
	require.Nil(t, err)
	require.Equal(t, access.CreatedAt.Unix(), result.CreatedAt.Unix())
	require.Equal(t, access.AccessData.CreatedAt.Unix(), result.AccessData.CreatedAt.Unix())
	require.Equal(t, access.AuthorizeData.CreatedAt.Unix(), result.AuthorizeData.CreatedAt.Unix())
	access.CreatedAt = result.CreatedAt
	access.AccessData.CreatedAt = result.AccessData.CreatedAt
	access.AuthorizeData.CreatedAt = result.AuthorizeData.CreatedAt
	require.Equal(t, access.UserData, result.UserData)

	require.Nil(t, store.RemoveAuthorize(authorize.Code))
	_, err = store.LoadAccess(access.AccessToken)
	require.Nil(t, err)

	require.Nil(t, store.RemoveAccess(nestedAccess.AccessToken))
	_, err = store.LoadAccess(access.AccessToken)
	require.Nil(t, err)

	require.Nil(t, store.RemoveAccess(access.AccessToken))
	_, err = store.LoadAccess(access.AccessToken)
	require.NotNil(t, err)

	require.Nil(t, store.RemoveAuthorize(authorize.Code))
	removeClient(t, store, client)
}

func TestRefreshOperations(t *testing.T) {
	client := &Client{Code: "4", Secret: "secret", RedirectUri: "http://localhost/", Meta: clientMetaEmpty}
	type test struct {
		access *osin.AccessData
	}

	for k, c := range []*test{
		{
			access: &osin.AccessData{
				Client: client,
				AuthorizeData: &osin.AuthorizeData{
					Client:      client,
					Code:        uuid.New(),
					ExpiresIn:   int32(60),
					Scope:       "scope",
					RedirectUri: "http://localhost/",
					State:       "state",
					CreatedAt:   time.Now().Round(time.Second),
					UserData:    userDataMock,
				},
				AccessData:   nil,
				AccessToken:  uuid.New(),
				RefreshToken: uuid.New(),
				ExpiresIn:    int32(60),
				Scope:        "scope",
				RedirectUri:  "https://localhost/",
				CreatedAt:    time.Now().Round(time.Second),
				UserData:     userDataMock,
			},
		},
	} {
		createClient(t, store, client)
		require.Nil(t, store.SaveAuthorize(c.access.AuthorizeData), "Case %d", k)
		require.Nil(t, store.SaveAccess(c.access), "Case %d", k)

		result, err := store.LoadRefresh(c.access.RefreshToken)
		require.Nil(t, err)
		require.Equal(t, c.access.CreatedAt.Unix(), result.CreatedAt.Unix())
		require.Equal(t, c.access.AuthorizeData.CreatedAt.Unix(), result.AuthorizeData.CreatedAt.Unix())
		c.access.CreatedAt = result.CreatedAt
		c.access.AuthorizeData.CreatedAt = result.AuthorizeData.CreatedAt
		require.Equal(t, c.access.AccessToken, result.AccessToken, "Case %d", k)

		require.Nil(t, store.RemoveRefresh(c.access.RefreshToken))
		_, err = store.LoadRefresh(c.access.RefreshToken)

		require.NotNil(t, err, "Case %d", k)
		require.Nil(t, store.RemoveAccess(c.access.AccessToken), "Case %d", k)
		require.Nil(t, store.SaveAccess(c.access), "Case %d", k)

		_, err = store.LoadRefresh(c.access.RefreshToken)
		require.Nil(t, err, "Case %d", k)

		require.Nil(t, store.RemoveAccess(c.access.AccessToken), "Case %d", k)
		_, err = store.LoadRefresh(c.access.RefreshToken)
		require.NotNil(t, err, "Case %d", k)

	}
	removeClient(t, store, client)
}

func TestErrors(t *testing.T) {
	client := &Client{Code: "dupe", Meta: clientMetaEmpty}
	assert.Nil(t, store.SaveClient(client))
	assert.NotNil(t, store.SaveClient(client))
	assert.NotNil(t, store.SaveClient(&Client{Code: "", Meta: clientMetaEmpty}))
	assert.NotNil(t, store.SaveAccess(&osin.AccessData{AccessToken: "", AccessData: &osin.AccessData{}, AuthorizeData: &osin.AuthorizeData{}}))
	assert.Nil(t, store.SaveAuthorize(&osin.AuthorizeData{Code: "a", Client: client, UserData: userDataMock}))
	assert.NotNil(t, store.SaveAuthorize(&osin.AuthorizeData{Code: "a", Client: client}))
	assert.NotNil(t, store.SaveAuthorize(&osin.AuthorizeData{Code: "b", Client: client}))
	_, err := store.LoadAccess("")
	assert.Equal(t, errNotFound, err)
	_, err = store.LoadAuthorize("")
	assert.Equal(t, errNotFound, err)
	_, err = store.LoadRefresh("")
	assert.Equal(t, errNotFound, err)
	_, err = store.GetClient("")
	assert.Equal(t, errNotFound, err)
}

func getClient(t *testing.T, store storage.Storage, set storage.Client) {
	client, err := store.GetClient(set.GetId())
	require.Nil(t, err)
	// require.EqualValues(t, set, client)
	require.Equal(t, set.GetId(), client.GetId())
	require.Equal(t, set.GetSecret(), client.GetSecret())
	require.Equal(t, set.GetRedirectUri(), client.GetRedirectUri())
	require.Equal(t, set.GetUserData(), client.GetUserData())
}

func createClient(t *testing.T, store storage.Storage, set storage.Client) {
	require.Nil(t, store.SaveClient(set))
}

func updateClient(t *testing.T, store storage.Storage, set storage.Client) {
	require.Nil(t, store.SaveClient(set))
}

func removeClient(t *testing.T, store storage.Storage, set storage.Client) {
	require.Nil(t, store.RemoveClient(set.GetId()))
}
