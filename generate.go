//go:generate sh -c "go run ./cmd/sqlitegen --header ${SQLITE_HEADER:-/usr/include/sqlite3.h} --output-dir ."
//go:generate mockery

package puregosqlite
