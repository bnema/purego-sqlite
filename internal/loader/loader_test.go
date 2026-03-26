package loader

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_Default(t *testing.T) {
	t.Setenv("SQLITE_LIB_PATH", "")
	handle, err := Open()
	require.NoError(t, err)
	assert.NotZero(t, handle)
}

func TestOpen_EnvOverride(t *testing.T) {
	t.Setenv("SQLITE_LIB_PATH", "/nonexistent/libsqlite3.so")
	_, err := Open()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "/nonexistent/libsqlite3.so")
}

func TestOpen_EnvOverrideValid(t *testing.T) {
	t.Setenv("SQLITE_LIB_PATH", "libsqlite3.so")
	handle, err := Open()
	require.NoError(t, err)
	assert.NotZero(t, handle)
}
