# osin-storage
A storage backend for [osin oauth2](https://github.com/RangelReale/osin) with [go-pg](https://github.com/go-pg/pg).


## Prepare database

```sh
cat storage/database/oauth_schema.sql | docker exec -i osin-db psql -U osin
``
