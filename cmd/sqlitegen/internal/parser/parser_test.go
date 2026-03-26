package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFunctions_Simple(t *testing.T) {
	input := []byte(`SQLITE_API int sqlite3_close(sqlite3*);`)
	h, err := Parse("test.h", input)
	require.NoError(t, err)
	require.Len(t, h.Functions, 1)

	f := h.Functions[0]
	assert.Equal(t, "sqlite3_close", f.CName)
	assert.Equal(t, "Sqlite3Close", f.GoName)
	assert.Equal(t, "int", f.ReturnCType)
	assert.Equal(t, "int32", f.ReturnGoType)
	assert.False(t, f.IsVariadic)
	require.Len(t, f.Params, 1)
	assert.Equal(t, "unsafe.Pointer", f.Params[0].GoType)
}

func TestParseFunctions_MultiParam(t *testing.T) {
	input := []byte(`SQLITE_API int sqlite3_open_v2(
  const char *filename,
  sqlite3 **ppDb,
  int flags,
  const char *zVfs
);`)
	h, err := Parse("test.h", input)
	require.NoError(t, err)
	require.Len(t, h.Functions, 1)

	f := h.Functions[0]
	assert.Equal(t, "sqlite3_open_v2", f.CName)
	assert.Equal(t, "Sqlite3OpenV2", f.GoName)
	assert.Equal(t, "int", f.ReturnCType)
	assert.Equal(t, "int32", f.ReturnGoType)
	require.Len(t, f.Params, 4)
	assert.Equal(t, "string", f.Params[0].GoType)
	assert.Equal(t, "filename", f.Params[0].CName)
	assert.Equal(t, "unsafe.Pointer", f.Params[1].GoType)
	assert.Equal(t, "int32", f.Params[2].GoType)
	assert.Equal(t, "string", f.Params[3].GoType)
}

func TestParseFunctions_Variadic(t *testing.T) {
	input := []byte(`SQLITE_API int sqlite3_config(int, ...);`)
	h, err := Parse("test.h", input)
	require.NoError(t, err)
	require.Len(t, h.Functions, 1)

	f := h.Functions[0]
	assert.Equal(t, "sqlite3_config", f.CName)
	assert.True(t, f.IsVariadic)
}

func TestParseDefines(t *testing.T) {
	input := []byte(`
#define SQLITE_OK           0
#define SQLITE_ERROR        1
#define SQLITE_OPEN_READONLY 0x00000001
#define SQLITE_VERSION      "3.52.0"
`)
	h, err := Parse("test.h", input)
	require.NoError(t, err)
	require.Len(t, h.Defines, 4)

	assert.Equal(t, "SQLITE_OK", h.Defines[0].CName)
	assert.Equal(t, "0", h.Defines[0].Value)
	assert.Equal(t, "SQLITE_ERROR", h.Defines[1].CName)
	assert.Equal(t, "1", h.Defines[1].Value)
	assert.Equal(t, "SQLITE_OPEN_READONLY", h.Defines[2].CName)
	assert.Equal(t, "0x00000001", h.Defines[2].Value)
	assert.Equal(t, "SQLITE_VERSION", h.Defines[3].CName)
	assert.Equal(t, `"3.52.0"`, h.Defines[3].Value)
}

func TestParseTypedefs(t *testing.T) {
	input := []byte(`
typedef struct sqlite3 sqlite3;
typedef struct sqlite3_stmt sqlite3_stmt;
typedef struct sqlite3_value sqlite3_value;
`)
	h, err := Parse("test.h", input)
	require.NoError(t, err)
	require.Len(t, h.Typedefs, 3)

	assert.Equal(t, "sqlite3", h.Typedefs[0].CName)
	assert.Equal(t, "Sqlite3", h.Typedefs[0].GoName)
	assert.Equal(t, "sqlite3_stmt", h.Typedefs[1].CName)
	assert.Equal(t, "Sqlite3Stmt", h.Typedefs[1].GoName)
}

func TestParseFunctions_VoidReturn(t *testing.T) {
	input := []byte(`SQLITE_API void sqlite3_free(void*);`)
	h, err := Parse("test.h", input)
	require.NoError(t, err)
	require.Len(t, h.Functions, 1)

	f := h.Functions[0]
	assert.Equal(t, "void", f.ReturnCType)
	assert.Equal(t, "", f.ReturnGoType)
}

func TestParseFunctions_CallbackParam(t *testing.T) {
	input := []byte(`SQLITE_API int sqlite3_exec(sqlite3*, const char *sql, int (*callback)(void*,int,char**,char**), void *, char **errmsg);`)
	h, err := Parse("test.h", input)
	require.NoError(t, err)
	require.Len(t, h.Functions, 1)

	f := h.Functions[0]
	assert.Equal(t, "sqlite3_exec", f.CName)
	require.Len(t, f.Params, 5)
	// callback param should be unsafe.Pointer
	assert.Equal(t, "unsafe.Pointer", f.Params[2].GoType)
}

func TestStripComments(t *testing.T) {
	input := "/* block comment */ int x; // line comment\nint y;"
	result := stripComments(input)
	assert.NotContains(t, result, "block comment")
	assert.NotContains(t, result, "line comment")
	assert.Contains(t, result, "int x;")
	assert.Contains(t, result, "int y;")
}

func TestParseRealHeader(t *testing.T) {
	h, err := ParseFile("/usr/include/sqlite3.h")
	require.NoError(t, err)

	t.Logf("Functions: %d", len(h.Functions))
	t.Logf("Defines:   %d", len(h.Defines))
	t.Logf("Typedefs:  %d", len(h.Typedefs))

	assert.GreaterOrEqual(t, len(h.Functions), 300, "expected 300+ functions")
	assert.GreaterOrEqual(t, len(h.Defines), 100, "expected 100+ defines")
	assert.GreaterOrEqual(t, len(h.Typedefs), 10, "expected 10+ typedefs")

	// Spot-check sqlite3_open_v2
	var openV2 *model_Function
	for i := range h.Functions {
		if h.Functions[i].CName == "sqlite3_open_v2" {
			f := h.Functions[i]
			openV2 = &f
			break
		}
	}
	require.NotNil(t, openV2, "sqlite3_open_v2 not found")
	assert.Equal(t, "Sqlite3OpenV2", openV2.GoName)
	assert.Equal(t, "int", openV2.ReturnCType)
	assert.Equal(t, "int32", openV2.ReturnGoType)
	assert.Len(t, openV2.Params, 4)
	assert.False(t, openV2.IsVariadic)
}
