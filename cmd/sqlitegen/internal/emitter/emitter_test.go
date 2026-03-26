package emitter

import (
	"go/format"
	"strings"
	"testing"
)

// testParam mirrors model.Param for template data.
type testParam struct {
	CName  string
	GoName string
	CType  string
	GoType string
}

// testFunction mirrors model.Function for template data.
type testFunction struct {
	CName        string
	GoName       string
	ReturnCType  string
	ReturnGoType string
	Params       []testParam
	IsVariadic   bool
}

// testDefine mirrors model.Define for template data.
type testDefine struct {
	CName string
	Value string
}

// testGroup mirrors a named group of defines for the enums template.
type testGroup struct {
	Name    string
	Defines []testDefine
}

// testArityArg is a placeholder element used to drive the arg range in variadic template.
type testArityArg struct{}

// testArity holds a single arity variant for a variadic function.
type testArity struct {
	Arity int
	Args  []testArityArg
}

// testVariadicFunction holds a variadic function with its arity variants.
type testVariadicFunction struct {
	CName   string
	GoName  string
	Arities []testArity
}

// isValidGo returns true if src is valid, gofmt-clean Go source.
func isValidGo(src string) bool {
	_, err := format.Source([]byte(src))
	return err == nil
}

func TestEmitFunctionsTemplate(t *testing.T) {
	data := struct {
		Functions []testFunction
	}{
		Functions: []testFunction{
			{
				CName:        "sqlite3_close",
				GoName:       "Sqlite3Close",
				ReturnCType:  "int",
				ReturnGoType: "int32",
				Params: []testParam{
					{CName: "db", GoName: "db", CType: "sqlite3*", GoType: "unsafe.Pointer"},
				},
				IsVariadic: false,
			},
			{
				CName:        "sqlite3_config",
				GoName:       "Sqlite3Config",
				ReturnCType:  "int",
				ReturnGoType: "int32",
				Params: []testParam{
					{CName: "", GoName: "", CType: "int", GoType: "int32"},
				},
				IsVariadic: true,
			},
		},
	}

	out, err := EmitTemplate("functions.tmpl", data)
	if err != nil {
		t.Fatalf("EmitTemplate failed: %v", err)
	}

	// Verify it contains the non-variadic var declaration.
	if !strings.Contains(out, "var Sqlite3Close func(") {
		t.Errorf("expected 'var Sqlite3Close func(' in output, got:\n%s", out)
	}

	// Verify the variadic function is excluded.
	if strings.Contains(out, "Sqlite3Config") {
		t.Errorf("variadic function Sqlite3Config should not appear in functions.tmpl output, got:\n%s", out)
	}

	// Verify param and return type are present.
	if !strings.Contains(out, "unsafe.Pointer") {
		t.Errorf("expected 'unsafe.Pointer' param type in output, got:\n%s", out)
	}
	if !strings.Contains(out, "int32") {
		t.Errorf("expected 'int32' return type in output, got:\n%s", out)
	}

	// Verify the output is valid Go.
	if !isValidGo(out) {
		t.Errorf("output is not valid Go:\n%s", out)
	}
}

func TestEmitFunctionsTemplate_VoidReturn(t *testing.T) {
	data := struct {
		Functions []testFunction
	}{
		Functions: []testFunction{
			{
				CName:        "sqlite3_free",
				GoName:       "Sqlite3Free",
				ReturnCType:  "void",
				ReturnGoType: "",
				Params: []testParam{
					{CName: "", GoName: "", CType: "void*", GoType: "unsafe.Pointer"},
				},
				IsVariadic: false,
			},
		},
	}

	out, err := EmitTemplate("functions.tmpl", data)
	if err != nil {
		t.Fatalf("EmitTemplate failed: %v", err)
	}

	if !strings.Contains(out, "var Sqlite3Free func(") {
		t.Errorf("expected 'var Sqlite3Free func(' in output, got:\n%s", out)
	}

	// No trailing return type for void functions.
	if strings.Contains(out, "Sqlite3Free func() ") {
		t.Errorf("void return should produce no return type, got:\n%s", out)
	}

	if !isValidGo(out) {
		t.Errorf("output is not valid Go:\n%s", out)
	}
}

func TestEmitEnumsTemplate(t *testing.T) {
	data := struct {
		Groups []testGroup
	}{
		Groups: []testGroup{
			{
				Name: "ResultCodes",
				Defines: []testDefine{
					{CName: "SQLITE_OK", Value: "0"},
					{CName: "SQLITE_ERROR", Value: "1"},
					{CName: "SQLITE_BUSY", Value: "5"},
				},
			},
			{
				Name: "OpenFlags",
				Defines: []testDefine{
					{CName: "SQLITE_OPEN_READONLY", Value: "0x00000001"},
					{CName: "SQLITE_OPEN_READWRITE", Value: "0x00000002"},
				},
			},
		},
	}

	out, err := EmitTemplate("enums.tmpl", data)
	if err != nil {
		t.Fatalf("EmitTemplate failed: %v", err)
	}

	if !strings.Contains(out, "SQLITE_OK") {
		t.Errorf("expected SQLITE_OK in output, got:\n%s", out)
	}
	if !strings.Contains(out, "SQLITE_OPEN_READONLY") {
		t.Errorf("expected SQLITE_OPEN_READONLY in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ResultCodes constants.") {
		t.Errorf("expected group comment 'ResultCodes constants.' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "const (") {
		t.Errorf("expected 'const (' in output, got:\n%s", out)
	}

	if !isValidGo(out) {
		t.Errorf("output is not valid Go:\n%s", out)
	}
}

func TestEmitRegisterTemplate(t *testing.T) {
	data := struct {
		Functions         []testFunction
		VariadicFunctions []testFunction
	}{
		Functions: []testFunction{
			{
				CName:      "sqlite3_close",
				GoName:     "Sqlite3Close",
				IsVariadic: false,
			},
			{
				CName:      "sqlite3_open",
				GoName:     "Sqlite3Open",
				IsVariadic: false,
			},
		},
		VariadicFunctions: []testFunction{
			{
				CName:  "sqlite3_config",
				GoName: "Sqlite3Config",
			},
		},
	}

	out, err := EmitTemplate("register.tmpl", data)
	if err != nil {
		t.Fatalf("EmitTemplate failed: %v", err)
	}

	// Verify RegisterLibFunc calls for non-variadic functions.
	if !strings.Contains(out, `purego.RegisterLibFunc(&Sqlite3Close, handle, "sqlite3_close")`) {
		t.Errorf("expected RegisterLibFunc call for Sqlite3Close, got:\n%s", out)
	}
	if !strings.Contains(out, `purego.RegisterLibFunc(&Sqlite3Open, handle, "sqlite3_open")`) {
		t.Errorf("expected RegisterLibFunc call for Sqlite3Open, got:\n%s", out)
	}

	// Verify Dlsym call for variadic function.
	if !strings.Contains(out, `purego.Dlsym(handle, "sqlite3_config")`) {
		t.Errorf("expected Dlsym call for sqlite3_config, got:\n%s", out)
	}

	// Verify the Register function signature.
	if !strings.Contains(out, "func Register(handle uintptr)") {
		t.Errorf("expected 'func Register(handle uintptr)' in output, got:\n%s", out)
	}

	if !isValidGo(out) {
		t.Errorf("output is not valid Go:\n%s", out)
	}
}

func TestEmitVariadicTemplate(t *testing.T) {
	data := struct {
		Functions []testVariadicFunction
	}{
		Functions: []testVariadicFunction{
			{
				CName:  "sqlite3_config",
				GoName: "Sqlite3Config",
				Arities: []testArity{
					{Arity: 1, Args: []testArityArg{{}}},
					{Arity: 2, Args: []testArityArg{{}, {}}},
				},
			},
		},
	}

	out, err := EmitTemplate("variadic.tmpl", data)
	if err != nil {
		t.Fatalf("EmitTemplate failed: %v", err)
	}

	if !strings.Contains(out, "var Sqlite3ConfigAddr uintptr") {
		t.Errorf("expected 'var Sqlite3ConfigAddr uintptr' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "func Sqlite3Config1(") {
		t.Errorf("expected 'func Sqlite3Config1(' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "func Sqlite3Config2(") {
		t.Errorf("expected 'func Sqlite3Config2(' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "purego.SyscallN(Sqlite3ConfigAddr") {
		t.Errorf("expected 'purego.SyscallN(Sqlite3ConfigAddr' in output, got:\n%s", out)
	}

	if !isValidGo(out) {
		t.Errorf("output is not valid Go:\n%s", out)
	}
}

func TestEmitTemplate_UnknownTemplate(t *testing.T) {
	_, err := EmitTemplate("nonexistent.tmpl", nil)
	if err == nil {
		t.Error("expected error for unknown template name, got nil")
	}
}
