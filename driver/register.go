package driver

import "database/sql"

func init() {
	sql.Register("sqlite3", &Driver{})
}
