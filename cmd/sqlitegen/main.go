// Command sqlitegen generates Go bindings for sqlite3.h using purego.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/bnema/purego-sqlite/cmd/sqlitegen/internal/emitter"
	"github.com/bnema/purego-sqlite/cmd/sqlitegen/internal/model"
	"github.com/bnema/purego-sqlite/cmd/sqlitegen/internal/overrides"
	"github.com/bnema/purego-sqlite/cmd/sqlitegen/internal/parser"
)

func main() {
	header := flag.String("header", "/usr/include/sqlite3.h", "path to sqlite3.h")
	outputDir := flag.String("output-dir", ".", "project root directory")
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("sqlitegen: ")

	// 1. Parse the header file.
	hdr, err := parser.ParseFile(*header)
	if err != nil {
		log.Fatalf("parse %s: %v", *header, err)
	}
	log.Printf("parsed %d functions, %d defines, %d typedefs",
		len(hdr.Functions), len(hdr.Defines), len(hdr.Typedefs))

	// 2. Sanitize parameter names (assign synthetic names to anonymous params,
	//    deduplicate, lowercase first letter for Go conventions).
	for i := range hdr.Functions {
		sanitizeParams(&hdr.Functions[i])
	}

	// 3. Apply overrides: separate regular, variadic, and skipped functions.
	var regular, variadic []model.Function
	for _, fn := range hdr.Functions {
		ovr := overrides.LookupOverride(fn.CName)
		if ovr != nil && ovr.Skip {
			continue
		}
		if fn.IsVariadic {
			variadic = append(variadic, fn)
		} else {
			regular = append(regular, fn)
		}
	}
	log.Printf("regular: %d, variadic: %d", len(regular), len(variadic))

	// 3. Group regular functions into outbound port groups.
	groups := groupFunctions(regular)
	log.Printf("port groups: %d", len(groups))

	// 4. Group #define constants into enum categories.
	enumGroups := groupDefines(hdr.Defines)
	log.Printf("enum groups: %d", len(enumGroups))

	// 5. Build variadic function data with arities.
	type variadicAritySlot struct {
		Arity int
		Args  []struct{} // len == Arity
	}
	type variadicFuncData struct {
		model.Function
		Arities []variadicAritySlot
	}
	var variadicData []variadicFuncData
	for _, fn := range variadic {
		ovr := overrides.LookupOverride(fn.CName)
		if ovr == nil || len(ovr.Arities) == 0 {
			continue
		}
		vd := variadicFuncData{Function: fn}
		for _, a := range ovr.Arities {
			slot := variadicAritySlot{Arity: a}
			slot.Args = make([]struct{}, a)
			vd.Arities = append(vd.Arities, slot)
		}
		variadicData = append(variadicData, vd)
	}

	// 6. Emit generated files.
	root := *outputDir

	// -- internal/capi/functions_gen.go
	emitFile(root, "internal/capi/functions_gen.go", "functions.tmpl", struct {
		Functions []model.Function
	}{Functions: regular})

	// -- internal/capi/enums_gen.go
	emitFile(root, "internal/capi/enums_gen.go", "enums.tmpl", struct {
		Groups []enumGroup
	}{Groups: enumGroups})

	// -- internal/capi/variadic_gen.go
	emitFile(root, "internal/capi/variadic_gen.go", "variadic.tmpl", struct {
		Functions []variadicFuncData
	}{Functions: variadicData})

	// -- internal/capi/register_gen.go
	emitFile(root, "internal/capi/register_gen.go", "register.tmpl", struct {
		Functions         []model.Function
		VariadicFunctions []model.Function
	}{Functions: regular, VariadicFunctions: variadic})

	// -- internal/ports/out/<group>_gen.go (one per group)
	var groupNames []string
	for _, g := range groups {
		needsUnsafe := groupNeedsUnsafe(g)
		emitFile(root, fmt.Sprintf("internal/ports/out/%s_gen.go", g.File), "port_out.tmpl", struct {
			Name        string
			Functions   []model.Function
			NeedsUnsafe bool
		}{Name: g.Name, Functions: g.Functions, NeedsUnsafe: needsUnsafe})
		groupNames = append(groupNames, g.Name)
	}

	// -- internal/ports/out/capi_gen.go (composite interface)
	emitCAPIInterface(root, groupNames)

	// -- internal/ports/in/ports_gen.go
	emitFile(root, "internal/ports/in/ports_gen.go", "port_in.tmpl", nil)

	// -- sqlite/sqlite_gen.go
	emitFile(root, "sqlite/sqlite_gen.go", "public_api.tmpl", nil)

	log.Println("done")
}

// groupFunctions assigns regular functions to port groups based on prefix matching.
func groupFunctions(fns []model.Function) []model.Group {
	assigned := make(map[string]bool)
	var result []model.Group

	for _, pg := range overrides.Groups {
		g := model.Group{Name: pg.Name, File: pg.File}
		for _, fn := range fns {
			if assigned[fn.CName] {
				continue
			}
			for _, prefix := range pg.Prefixes {
				if strings.HasPrefix(fn.CName, prefix) {
					g.Functions = append(g.Functions, fn)
					assigned[fn.CName] = true
					break
				}
			}
		}
		if len(g.Functions) > 0 {
			result = append(result, g)
		}
	}

	// Catch-all Misc group for unmatched functions.
	var misc model.Group
	misc.Name = "Misc"
	misc.File = "misc"
	for _, fn := range fns {
		if !assigned[fn.CName] {
			misc.Functions = append(misc.Functions, fn)
		}
	}
	if len(misc.Functions) > 0 {
		result = append(result, misc)
	}

	return result
}

// enumGroup holds a named group of #define constants.
type enumGroup struct {
	Name    string
	Defines []model.Define
}

// isValidGoConstValue checks if a #define value can be used as a Go constant.
func isValidGoConstValue(val string) bool {
	// Skip C casts like ((sqlite3_destructor_type)0)
	if strings.Contains(val, "sqlite3_") && strings.Contains(val, ")") {
		return false
	}
	// Skip empty values
	if val == "" {
		return false
	}
	return true
}

// groupDefines groups #define constants by their prefix category.
func groupDefines(defines []model.Define) []enumGroup {
	// Build a map of prefix -> defines.
	// The category is derived from the define name by stripping the last
	// component. E.g. SQLITE_OK -> "Result", SQLITE_OPEN_READONLY -> "Open".
	groupMap := make(map[string][]model.Define)

	for _, d := range defines {
		if !isValidGoConstValue(d.Value) {
			continue
		}
		cat := defineCategory(d.CName)
		groupMap[cat] = append(groupMap[cat], d)
	}

	// Sort category names for deterministic output.
	var cats []string
	for c := range groupMap {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	var result []enumGroup
	for _, c := range cats {
		result = append(result, enumGroup{Name: c, Defines: groupMap[c]})
	}
	return result
}

// defineCategory extracts a category name from a SQLITE_ define name.
// SQLITE_OK -> "Result", SQLITE_OPEN_READONLY -> "Open", etc.
func defineCategory(name string) string {
	// Strip the SQLITE_ prefix.
	trimmed := strings.TrimPrefix(name, "SQLITE_")
	if trimmed == name {
		return "Other"
	}

	// Known result codes (single-word defines): map to "Result".
	resultCodes := map[string]bool{
		"OK": true, "ERROR": true, "INTERNAL": true, "PERM": true,
		"ABORT": true, "BUSY": true, "LOCKED": true, "NOMEM": true,
		"READONLY": true, "INTERRUPT": true, "IOERR": true, "CORRUPT": true,
		"NOTFOUND": true, "FULL": true, "CANTOPEN": true, "PROTOCOL": true,
		"EMPTY": true, "SCHEMA": true, "TOOBIG": true, "CONSTRAINT": true,
		"MISMATCH": true, "MISUSE": true, "NOLFS": true, "AUTH": true,
		"FORMAT": true, "RANGE": true, "NOTADB": true, "NOTICE": true,
		"WARNING": true, "ROW": true, "DONE": true,
	}

	// Split by underscore; use first word as category.
	parts := strings.SplitN(trimmed, "_", 2)
	first := parts[0]

	if resultCodes[first] && len(parts) == 1 {
		return "Result"
	}

	// Extended result codes: SQLITE_IOERR_READ -> "Ioerr"
	if resultCodes[first] && len(parts) == 2 {
		return first[0:1] + strings.ToLower(first[1:])
	}

	return first[0:1] + strings.ToLower(first[1:])
}

// groupNeedsUnsafe checks if any function in a group uses unsafe.Pointer.
func groupNeedsUnsafe(g model.Group) bool {
	for _, fn := range g.Functions {
		if fn.ReturnGoType == "unsafe.Pointer" {
			return true
		}
		for _, p := range fn.Params {
			if p.GoType == "unsafe.Pointer" {
				return true
			}
		}
	}
	return false
}

// emitFile renders a template with data and writes the result to outPath.
func emitFile(root, relPath, tmplName string, data any) {
	out, err := emitter.EmitTemplate(tmplName, data)
	if err != nil {
		log.Fatalf("emit %s: %v", relPath, err)
	}
	absPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", filepath.Dir(absPath), err)
	}
	if err := os.WriteFile(absPath, []byte(out), 0o644); err != nil {
		log.Fatalf("write %s: %v", absPath, err)
	}
	log.Printf("wrote %s (%d bytes)", relPath, len(out))
}

// goKeywords contains Go keywords and predeclared identifiers that cannot
// be used as parameter names.
var goKeywords = map[string]bool{
	// Keywords.
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
	// Predeclared types that cause confusion as param names.
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"uintptr": true, "float32": true, "float64": true, "complex64": true, "complex128": true,
	"string": true, "bool": true, "byte": true, "rune": true, "error": true,
	"any": true, "comparable": true,
	// C types that leak through as names.
	"double": true,
}

// sanitizeParams ensures all params have valid Go names and no duplicates.
func sanitizeParams(fn *model.Function) {
	// Assign synthetic names to unnamed params or params with keyword names.
	unnamed := 0
	for i := range fn.Params {
		name := fn.Params[i].GoName
		if name == "" || goKeywords[strings.ToLower(name)] || goKeywords[name] {
			fn.Params[i].GoName = fmt.Sprintf("arg%d", unnamed)
			unnamed++
		} else {
			// Lowercase the first letter for Go param convention.
			fn.Params[i].GoName = lowerFirst(fn.Params[i].GoName)
		}
	}

	// Deduplicate param names by appending a numeric suffix.
	seen := make(map[string]int)
	for i := range fn.Params {
		name := fn.Params[i].GoName
		if count, ok := seen[name]; ok {
			fn.Params[i].GoName = fmt.Sprintf("%s%d", name, count)
			seen[name] = count + 1
		} else {
			seen[name] = 1
		}
	}
}

// lowerFirst lowercases the first character of a string.
func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

// emitCAPIInterface generates the composite CAPI interface via string builder.
func emitCAPIInterface(root string, groupNames []string) {
	var sb strings.Builder
	sb.WriteString("// Code generated by sqlitegen. DO NOT EDIT.\npackage out\n\n")
	sb.WriteString("// CAPI composes all outbound port interfaces.\ntype CAPI interface {\n")
	for _, name := range groupNames {
		sb.WriteString("\t")
		sb.WriteString(name)
		sb.WriteString("\n")
	}
	sb.WriteString("}\n")

	absPath := filepath.Join(root, "internal/ports/out/capi_gen.go")
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(sb.String()), 0o644); err != nil {
		log.Fatalf("write capi_gen.go: %v", err)
	}
	log.Printf("wrote internal/ports/out/capi_gen.go (%d bytes)", sb.Len())
}
