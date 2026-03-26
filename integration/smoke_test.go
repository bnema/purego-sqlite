//go:build integration

package integration

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/bnema/purego-sqlite/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmoke_OpenClose(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	require.NoError(t, db.Close())
}

func TestSmoke_InMemory(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM t").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSmoke_CreateInsertQuery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT)")
	require.NoError(t, err)

	_, err = db.Exec("INSERT INTO t (name) VALUES (?)", "hello")
	require.NoError(t, err)

	var name string
	err = db.QueryRow("SELECT name FROM t WHERE id = 1").Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "hello", name)
}

func TestSmoke_Transaction(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, val TEXT)")
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)

	_, err = tx.Exec("INSERT INTO t (val) VALUES (?)", "inside-tx")
	require.NoError(t, err)

	require.NoError(t, tx.Commit())

	var val string
	err = db.QueryRow("SELECT val FROM t WHERE id = 1").Scan(&val)
	require.NoError(t, err)
	assert.Equal(t, "inside-tx", val)
}

func TestSmoke_Rollback(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, val TEXT)")
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)

	_, err = tx.Exec("INSERT INTO t (val) VALUES (?)", "will-rollback")
	require.NoError(t, err)

	require.NoError(t, tx.Rollback())

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM t").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSmoke_MultipleRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, val INTEGER)")
	require.NoError(t, err)

	for i := 1; i <= 100; i++ {
		_, err = db.Exec("INSERT INTO t (val) VALUES (?)", i)
		require.NoError(t, err)
	}

	rows, err := db.Query("SELECT val FROM t ORDER BY val")
	require.NoError(t, err)
	defer rows.Close()

	var vals []int
	for rows.Next() {
		var v int
		require.NoError(t, rows.Scan(&v))
		vals = append(vals, v)
	}
	require.NoError(t, rows.Err())
	assert.Len(t, vals, 100)
	assert.Equal(t, 1, vals[0])
	assert.Equal(t, 100, vals[99])
}

func TestSmoke_EnvOverride(t *testing.T) {
	t.Setenv("SQLITE_LIB_PATH", "libsqlite3.so")
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Ping())
	require.NoError(t, db.Close())
}
