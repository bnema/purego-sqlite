// Package parser extracts SQLITE_API function declarations, #define constants,
// and typedef structs from sqlite3.h.
package parser

import (
	"os"
	"regexp"
	"strings"
	"unicode"

	"github.com/bnema/purego-sqlite/cmd/sqlitegen/internal/model"
)

// model_Function is a local alias used in tests.
type model_Function = model.Function

var (
	// funcRE matches SQLITE_API [SQLITE_DEPRECATED] <return> [*] <name>(<params>);
	// on a single joined line.  Group 1 = base return type, group 2 = optional *,
	// group 3 = C function name, group 4 = raw params.
	// The name may be sqlite3_* or sqlite3session_* etc. (any sqlite3 prefix).
	funcRE = regexp.MustCompile(
		`^SQLITE_API\s+(?:SQLITE_DEPRECATED\s+)?(?:SQLITE_EXPERIMENTAL\s+)?([\w\s\*]+?)\s*(\*?)\s*(sqlite3[\w]*)\s*\(([^)]*(?:\([^)]*\)[^)]*)*)\)\s*;$`,
	)

	// defineRE matches #define SQLITE_NAME value
	// Captures: simple tokens, parenthesized expressions, and quoted strings.
	// Excludes: function-like macros (name followed by '(') and backslash continuations.
	defineRE = regexp.MustCompile(`^#define\s+(SQLITE_\w+)\s+(\(.*\)|"[^"]*"|[^\s(\\]+)`)

	// typedefRE matches typedef struct name name;
	typedefRE = regexp.MustCompile(`^typedef\s+struct\s+(\w+)\s+(\w+)\s*;$`)
)

// ParseFile reads the file at path then calls Parse.
func ParseFile(path string) (*model.Header, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, data)
}

// Parse extracts functions, defines, and typedefs from sqlite3.h content.
func Parse(path string, data []byte) (*model.Header, error) {
	src := string(data)

	src = stripComments(src)
	lines := joinLines(src)

	h := &model.Header{Path: path}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Try function
		if f, ok := parseFunction(line); ok {
			h.Functions = append(h.Functions, f)
			continue
		}

		// Try define
		if d, ok := parseDefine(line); ok {
			h.Defines = append(h.Defines, d)
			continue
		}

		// Try typedef
		if td, ok := parseTypedef(line); ok {
			h.Typedefs = append(h.Typedefs, td)
		}
	}

	return h, nil
}

// stripComments removes block /* */ and line // comments from src.
func stripComments(src string) string {
	// Remove block comments
	var b strings.Builder
	b.Grow(len(src))
	i := 0
	for i < len(src) {
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '*' {
			// Find closing */
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				break
			}
			// Preserve newlines so line numbers stay roughly correct
			for _, c := range src[i : i+2+end+2] {
				if c == '\n' {
					b.WriteByte('\n')
				}
			}
			i = i + 2 + end + 2
			continue
		}
		if i+1 < len(src) && src[i] == '/' && src[i+1] == '/' {
			// Find end of line
			end := strings.IndexByte(src[i:], '\n')
			if end < 0 {
				break
			}
			i = i + end // keep the newline
			continue
		}
		b.WriteByte(src[i])
		i++
	}
	return b.String()
}

// isCompleteLine reports whether a preprocessor or non-function line stands alone.
func isCompleteLine(s string) bool {
	// Preprocessor directives are always complete on their own line.
	if strings.HasPrefix(s, "#") {
		return true
	}
	return false
}

// joinLines collapses multi-line declarations into single lines.
// Lines ending with ; { } are statement terminators.
// Preprocessor lines (#define, #ifdef, etc.) always stand alone.
// Other lines that don't end with ; { } are continuations of the next line.
func joinLines(src string) []string {
	rawLines := strings.Split(src, "\n")
	var result []string
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			result = append(result, current.String())
			current.Reset()
		}
	}

	for _, line := range rawLines {
		trimmed := strings.TrimRight(line, " \t\r")
		stripped := strings.TrimSpace(trimmed)

		if stripped == "" {
			flush()
			continue
		}

		// Preprocessor directives always stand alone.
		if isCompleteLine(stripped) {
			flush()
			result = append(result, stripped)
			continue
		}

		if current.Len() > 0 {
			current.WriteByte(' ')
		}
		current.WriteString(stripped)

		last := stripped[len(stripped)-1]
		if last == ';' || last == '{' || last == '}' {
			flush()
		}
	}
	flush()
	return result
}

// parseFunction tries to parse a SQLITE_API function declaration from a single line.
func parseFunction(line string) (model.Function, bool) {
	if !strings.HasPrefix(line, "SQLITE_API") {
		return model.Function{}, false
	}

	m := funcRE.FindStringSubmatch(line)
	if m == nil {
		return model.Function{}, false
	}

	// m[1] = base return type, m[2] = optional *, m[3] = C name, m[4] = params
	retC := strings.TrimSpace(m[1]) + m[2]
	cname := strings.TrimSpace(m[3])
	rawParams := strings.TrimSpace(m[4])

	params, isVariadic := parseParams(rawParams)

	return model.Function{
		CName:        cname,
		GoName:       goName(cname),
		ReturnCType:  retC,
		ReturnGoType: mapType(retC),
		Params:       params,
		IsVariadic:   isVariadic,
	}, true
}

// parseDefine tries to parse a #define SQLITE_... constant.
func parseDefine(line string) (model.Define, bool) {
	if !strings.HasPrefix(line, "#define") {
		return model.Define{}, false
	}

	m := defineRE.FindStringSubmatch(line)
	if m == nil {
		return model.Define{}, false
	}

	return model.Define{
		CName: m[1],
		Value: m[2],
	}, true
}

// parseTypedef tries to parse a typedef struct declaration.
func parseTypedef(line string) (model.Typedef, bool) {
	m := typedefRE.FindStringSubmatch(line)
	if m == nil {
		return model.Typedef{}, false
	}

	cname := m[2]
	return model.Typedef{
		CName:  cname,
		GoName: goName(cname),
	}, true
}

// parseParams splits a raw parameter string into model.Param slices.
// It handles nested parens (callback function pointers) and detects variadic.
func parseParams(raw string) ([]model.Param, bool) {
	if strings.TrimSpace(raw) == "" || strings.TrimSpace(raw) == "void" {
		return nil, false
	}

	parts := splitParams(raw)
	isVariadic := false
	var params []model.Param

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if part == "..." {
			isVariadic = true
			continue
		}

		p := parseSingleParam(part)
		params = append(params, p)
	}

	return params, isVariadic
}

// splitParams splits a parameter list by commas, respecting nested parentheses.
func splitParams(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i, c := range s {
		switch c {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// parseSingleParam parses a single C parameter declaration into a model.Param.
func parseSingleParam(param string) model.Param {
	param = strings.TrimSpace(param)

	// Detect callback function pointer: type (*name)(args) or just (*)(...)
	if strings.Contains(param, "(*") {
		return model.Param{
			CType:  param,
			GoType: "unsafe.Pointer",
		}
	}

	// Count and strip stars
	pointers := strings.Count(param, "*")

	// Normalize: collapse multiple spaces
	param = strings.Join(strings.Fields(param), " ")

	// Split into type and name
	// The name is the last word (if it's an identifier), otherwise anonymous
	ctype, cname := splitTypeAndName(param)

	isConst := strings.Contains(ctype, "const")
	goType := mapType(ctype)

	goN := ""
	if cname != "" {
		goN = goName(cname)
	}

	return model.Param{
		CName:   cname,
		GoName:  goN,
		CType:   ctype,
		GoType:  goType,
		IsConst: isConst,
		Pointer: pointers,
	}
}

// splitTypeAndName splits a C declaration like "const char *filename" into
// type "const char*" and name "filename". Returns empty name for anonymous params.
//
// Examples:
//
//	"const char *filename"  -> type="const char*", name="filename"
//	"sqlite3*"              -> type="sqlite3*",    name=""
//	"sqlite3 **ppDb"        -> type="sqlite3**",   name="ppDb"
//	"int flags"             -> type="int",          name="flags"
//	"void *"                -> type="void*",        name=""
func splitTypeAndName(s string) (ctype, name string) {
	s = strings.TrimSpace(s)

	words := strings.Fields(s)
	if len(words) == 0 {
		return s, ""
	}

	last := words[len(words)-1]

	// Strip leading stars from the last word to find the identifier part.
	namePart := strings.TrimLeft(last, "*")

	if namePart != "" && (namePart[0] == '_' || unicode.IsLetter(rune(namePart[0]))) {
		// Check that namePart does NOT itself end with '*' (e.g. "sqlite3*" where
		// the star trails the type-name token). If it does, treat as anonymous.
		if !strings.Contains(namePart, "*") {
			// Leading stars were on the name token (pointer-to-name syntax)
			starsOnName := len(last) - len(namePart)
			typeWords := words[:len(words)-1]
			typePart := strings.Join(typeWords, " ")
			if starsOnName > 0 {
				typePart += strings.Repeat("*", starsOnName)
			}
			return typePart, namePart
		}
	}

	// Anonymous param (e.g. "sqlite3*", "void *", "int"): the whole thing is the type.
	// Normalize stars: strip trailing spaces before them and collapse.
	return s, ""
}

// mapType maps a C type string to a Go type string.
func mapType(ctype string) string {
	ctype = strings.TrimSpace(ctype)

	// Normalize spaces
	ctype = strings.Join(strings.Fields(ctype), " ")

	switch ctype {
	case "void":
		return ""
	case "void*", "void *", "const void*", "const void *":
		return "unsafe.Pointer"
	case "int":
		return "int32"
	case "unsigned int":
		return "uint32"
	case "const char*", "const char *", "char*", "char *":
		return "string"
	case "const unsigned char*", "const unsigned char *", "unsigned char*", "unsigned char *":
		return "*byte"
	case "sqlite3_int64", "sqlite3_int64*":
		return "int64"
	case "sqlite3_uint64":
		return "uint64"
	case "double":
		return "float64"
	case "size_t":
		return "uintptr"
	case "sqlite3_int64 *", "*sqlite3_int64":
		return "unsafe.Pointer"
	}

	// Any pointer type -> unsafe.Pointer
	if strings.Contains(ctype, "*") {
		return "unsafe.Pointer"
	}

	// Unknown non-pointer
	return "uintptr"
}

// goName converts a C identifier to Go style (PascalCase).
// e.g. sqlite3_open_v2 -> Sqlite3OpenV2
// Special tokens: id->ID, url->URL, sql->SQL, db->DB, vfs->VFS, wal->WAL, api->API
func goName(name string) string {
	// Replace dots and hyphens with underscores
	name = strings.NewReplacer(".", "_", "-", "_").Replace(name)

	parts := strings.Split(name, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(titleToken(p))
	}
	return b.String()
}

// titleToken converts a single token to title case, with special cases.
func titleToken(token string) string {
	upper := strings.ToUpper(token)
	switch upper {
	case "ID":
		return "ID"
	case "URL":
		return "URL"
	case "SQL":
		return "SQL"
	case "DB":
		return "DB"
	case "VFS":
		return "VFS"
	case "WAL":
		return "WAL"
	case "API":
		return "API"
	case "V2", "V3":
		return upper
	}
	// Title case: first char upper, rest lower
	if len(token) == 0 {
		return token
	}
	runes := []rune(token)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
