# osin-storage
A storage backend for [osin oauth2](https://github.com/RangelReale/osin) with:

* [go-pg](https://github.com/go-pg/pg).
* `sqlstore`: [pq](https://github.com/lib/pq) and [sqlx](https://github.com/jmoiron/sqlx)

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
