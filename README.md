# purego-sqlite

Pure Go SQLite bindings via [purego](https://github.com/bnema/purego). No cgo required.

Dynamically links to your system's `libsqlite3.so` at runtime.

## Install

    go get github.com/bnema/purego-sqlite

## Usage

### database/sql driver

    import (
        "database/sql"
        _ "github.com/bnema/purego-sqlite/driver"
    )

    db, err := sql.Open("sqlite3", "./my.db")

### Direct API

    import "github.com/bnema/purego-sqlite/sqlite"

    db, err := sqlite.Open("./my.db")

## Requirements

- `libsqlite3.so` installed on the system (present on virtually all Linux distros)
- Override path: `SQLITE_LIB_PATH=/custom/path/libsqlite3.so`

## License

MIT
