package driver

import (
	"database/sql/driver"
	"testing"

	"github.com/bnema/purego-sqlite/sqlite/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDriver_ImplementsInterface verifies compile-time interface satisfaction.
func TestDriver_ImplementsInterface(t *testing.T) {
	var _ driver.Driver = (*Driver)(nil)
	var _ driver.Conn = (*conn)(nil)
	var _ driver.Stmt = (*stmt)(nil)
	var _ driver.Rows = (*driverRows)(nil)
	var _ driver.Tx = (*tx)(nil)
}

// TestConn_Prepare verifies that conn.Prepare delegates to DB.Prepare.
func TestConn_Prepare(t *testing.T) {
	mockDB := mocks.NewMockDB(t)
	mockStmt := mocks.NewMockStmt(t)

	mockDB.EXPECT().Prepare("SELECT 1").Return(mockStmt, nil)

	c := &conn{db: mockDB}
	s, err := c.Prepare("SELECT 1")
	require.NoError(t, err)
	assert.NotNil(t, s)
}

// TestConn_BeginCommit verifies that Begin issues BEGIN and Commit issues COMMIT.
func TestConn_BeginCommit(t *testing.T) {
	mockDB := mocks.NewMockDB(t)

	mockDB.EXPECT().Exec("BEGIN").Return(nil, nil)
	mockDB.EXPECT().Exec("COMMIT").Return(nil, nil)

	c := &conn{db: mockDB}
	tx, err := c.Begin()
	require.NoError(t, err)
	require.NotNil(t, tx)

	err = tx.Commit()
	assert.NoError(t, err)
}

// TestConn_BeginRollback verifies that Begin issues BEGIN and Rollback issues ROLLBACK.
func TestConn_BeginRollback(t *testing.T) {
	mockDB := mocks.NewMockDB(t)

	mockDB.EXPECT().Exec("BEGIN").Return(nil, nil)
	mockDB.EXPECT().Exec("ROLLBACK").Return(nil, nil)

	c := &conn{db: mockDB}
	tx, err := c.Begin()
	require.NoError(t, err)
	require.NotNil(t, tx)

	err = tx.Rollback()
	assert.NoError(t, err)
}
