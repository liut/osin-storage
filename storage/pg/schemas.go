package pg

var tables = []string{"oauth.client", "oauth.access", "oauth.refresh", "oauth.authorize"}
var schemas = []string{
	"CREATE SCHEMA IF NOT EXISTS oauth",
}
