package core

import "fmt"

// SQLite result codes.
const (
	sqliteOK   = 0
	sqliteRow  = 100
	sqliteDone = 101
)

// SQLite column type codes.
const (
	sqliteInteger = 1
	sqliteFloat   = 2
	sqliteText    = 3
	sqliteBlob    = 4
	sqliteNull    = 5
)

// SQLite open flags.
const (
	sqliteOpenReadWrite = 0x00000002
	sqliteOpenCreate    = 0x00000004
)

// Error represents a SQLite error with a result code and message.
type Error struct {
	Code int
	Msg  string
}

func (e *Error) Error() string {
	return fmt.Sprintf("sqlite3: %s (%d)", e.Msg, e.Code)
}
