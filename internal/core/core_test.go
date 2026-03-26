package core

import (
	"testing"

	portin "github.com/bnema/purego-sqlite/internal/ports/in"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time interface satisfaction checks.
var (
	_ portin.DB     = (*database)(nil)
	_ portin.Stmt   = (*statement)(nil)
	_ portin.Rows   = (*rows)(nil)
	_ portin.Result = (*result)(nil)
)

func TestErrorString(t *testing.T) {
	e := &Error{Code: 1, Msg: "SQL logic error"}
	assert.Equal(t, "sqlite3: SQL logic error (1)", e.Error())
}

func TestErrorInterface(t *testing.T) {
	var err error = &Error{Code: 14, Msg: "unable to open database file"}
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sqlite3:")
	assert.Contains(t, err.Error(), "14")
}

func TestResultValues(t *testing.T) {
	r := &result{lastID: 42, changes: 3}

	lastID, err := r.LastInsertId()
	require.NoError(t, err)
	assert.Equal(t, int64(42), lastID)

	affected, err := r.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(3), affected)
}

func TestResultZero(t *testing.T) {
	r := &result{}

	lastID, err := r.LastInsertId()
	require.NoError(t, err)
	assert.Equal(t, int64(0), lastID)

	affected, err := r.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected)
}

func TestConstants(t *testing.T) {
	// Verify our constants match standard SQLite values.
	assert.Equal(t, 0, sqliteOK)
	assert.Equal(t, 100, sqliteRow)
	assert.Equal(t, 101, sqliteDone)

	assert.Equal(t, 1, sqliteInteger)
	assert.Equal(t, 2, sqliteFloat)
	assert.Equal(t, 3, sqliteText)
	assert.Equal(t, 4, sqliteBlob)
	assert.Equal(t, 5, sqliteNull)

	assert.Equal(t, 0x00000002, sqliteOpenReadWrite)
	assert.Equal(t, 0x00000004, sqliteOpenCreate)
}
