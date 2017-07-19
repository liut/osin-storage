# osin-storage
A storage backend for [osin oauth2](https://github.com/RangelReale/osin) with:

* `storage/pg`: [go-pg](https://github.com/go-pg/pg).
* `storage/sqlstore`: [pq](https://github.com/lib/pq) and [sqlx](https://github.com/jmoiron/sqlx)

This project was inspired from [ory-am](https://github.com/ory-am/osin-storage)

## Addition features

* Save map and struct meta information with `JSON`(or `JSONB`) for Client and Authorization
* Use `SaveClient()` instead of `CreateClient()` and `UpdateClient()`
* Add `AllClients() []` interface for management
* Add remember function for authorization

## Prepare database

```sh
cat storage/database/oauth_schema.sql | docker exec -i osin-db psql -U osin
```

## Example

```go

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/liut/osin-storage/storage/sqlstore"
	"github.com/RangelReale/osin"
)

func main () {
	dsn := "postgres://osin:osin@127.0.0.1:5432/osin?sslmode=disable"
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	store := sqlstore.New(db)
	server := osin.NewServer(newOsinConfig(), store)
}

```
