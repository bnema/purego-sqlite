package driver

import (
	"database/sql/driver"
	"io"

	"github.com/bnema/purego-sqlite/sqlite"
)

// Driver implements database/sql/driver.Driver.
type Driver struct{}

// Open opens a new SQLite database connection.
func (d *Driver) Open(dsn string) (driver.Conn, error) {
	db, err := sqlite.Open(dsn)
	if err != nil {
		return nil, err
	}
	return &conn{db: db}, nil
}

// conn implements database/sql/driver.Conn.
type conn struct {
	db sqlite.DB
}

// Prepare returns a prepared statement.
func (c *conn) Prepare(query string) (driver.Stmt, error) {
	s, err := c.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	return &stmt{s: s}, nil
}

// Close closes the connection.
func (c *conn) Close() error {
	return c.db.Close()
}

// Begin starts a transaction.
func (c *conn) Begin() (driver.Tx, error) {
	if _, err := c.db.Exec("BEGIN"); err != nil {
		return nil, err
	}
	return &tx{db: c.db}, nil
}

// tx implements database/sql/driver.Tx.
type tx struct {
	db sqlite.DB
}

// Commit commits the transaction.
func (t *tx) Commit() error {
	_, err := t.db.Exec("COMMIT")
	return err
}

// Rollback rolls back the transaction.
func (t *tx) Rollback() error {
	_, err := t.db.Exec("ROLLBACK")
	return err
}

// stmt implements database/sql/driver.Stmt.
type stmt struct {
	s sqlite.Stmt
}

// Close closes the statement.
func (s *stmt) Close() error {
	return s.s.Close()
}

// NumInput returns the number of placeholder parameters.
func (s *stmt) NumInput() int {
	return s.s.NumInput()
}

// Exec executes the statement with the given args.
func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	anyArgs := driverValuesToAny(args)
	return s.s.Exec(anyArgs...)
}

// Query executes the statement and returns rows.
func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	anyArgs := driverValuesToAny(args)
	rows, err := s.s.Query(anyArgs...)
	if err != nil {
		return nil, err
	}
	return &driverRows{rows: rows}, nil
}

// driverRows implements database/sql/driver.Rows.
type driverRows struct {
	rows sqlite.Rows
}

// Columns returns the column names.
func (r *driverRows) Columns() []string {
	cols, _ := r.rows.Columns()
	return cols
}

// Close closes the rows.
func (r *driverRows) Close() error {
	return r.rows.Close()
}

// Next fills dest with the next row's values.
// Returns io.EOF when there are no more rows.
func (r *driverRows) Next(dest []driver.Value) error {
	if !r.rows.Next() {
		return io.EOF
	}
	// We cannot pass *driver.Value to core.Scan because driver.Value is a
	// distinct named type (not a type alias) in Go 1.22+, so *driver.Value
	// does not match *any in a type switch. Use intermediate *any variables.
	vals := make([]any, len(dest))
	ptrs := make([]any, len(dest))
	for i := range dest {
		ptrs[i] = &vals[i]
	}
	if err := r.rows.Scan(ptrs...); err != nil {
		return err
	}
	for i := range dest {
		dest[i] = vals[i]
	}
	return nil
}

// driverValuesToAny converts a []driver.Value to []any.
func driverValuesToAny(args []driver.Value) []any {
	out := make([]any, len(args))
	for i, v := range args {
		out[i] = v
	}
	return out
}
