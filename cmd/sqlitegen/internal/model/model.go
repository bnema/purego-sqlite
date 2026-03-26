package model

// Header represents the parsed content of sqlite3.h.
type Header struct {
	Path      string
	Functions []Function
	Defines   []Define
	Typedefs  []Typedef
}

// Function represents a parsed SQLITE_API function declaration.
type Function struct {
	CName        string // e.g. "sqlite3_open_v2"
	GoName       string // e.g. "Sqlite3OpenV2"
	ReturnCType  string // e.g. "int"
	ReturnGoType string // e.g. "int32"
	Params       []Param
	IsVariadic   bool
}

// Param represents a function parameter.
type Param struct {
	CName   string // e.g. "filename"
	GoName  string // e.g. "filename"
	CType   string // e.g. "const char*"
	GoType  string // e.g. "string"
	IsConst bool
	Pointer int // number of * in the type
}

// Define represents a #define constant.
type Define struct {
	CName string // e.g. "SQLITE_OK"
	Value string // e.g. "0"
}

// Typedef represents a typedef struct declaration.
type Typedef struct {
	CName  string // e.g. "sqlite3"
	GoName string // e.g. "Sqlite3"
}

// Group holds parsed functions assigned to a specific outbound port group.
type Group struct {
	Name      string     // Go interface name: "Lifecycle", "Column"
	File      string     // output filename stem: "lifecycle", "column"
	Functions []Function // functions belonging to this group
}
