package overrides

// Override controls per-function generation behavior.
type Override struct {
	Name    string
	Skip    bool  // skip generation entirely
	Arities []int // for variadic: typed wrappers at these arities
}

// Overrides lists per-function generation overrides.
var Overrides = []Override{
	{Name: "sqlite3_version", Skip: true},
	{Name: "sqlite3_config", Arities: []int{1, 2, 3}},
	{Name: "sqlite3_db_config", Arities: []int{2, 3, 4}},
	{Name: "sqlite3_mprintf", Skip: true},
	{Name: "sqlite3_snprintf", Arities: []int{3, 4, 5}},
	{Name: "sqlite3_test_control", Arities: []int{1, 2, 3}},
	{Name: "sqlite3_str_appendf", Arities: []int{2, 3, 4}},
	{Name: "sqlite3_log", Arities: []int{2, 3}},
	{Name: "sqlite3_vtab_config", Arities: []int{2, 3}},
}

// PortGroup defines how C functions are grouped into outbound port interfaces.
type PortGroup struct {
	Name     string   // Go interface name
	File     string   // output filename stem
	Prefixes []string // C function prefixes
}

// Groups defines the prefix-based grouping for outbound port interfaces.
// Functions not matching any group go into a "Misc" catch-all.
var Groups = []PortGroup{
	{Name: "Lifecycle", File: "lifecycle", Prefixes: []string{
		"sqlite3_open", "sqlite3_close", "sqlite3_initialize", "sqlite3_shutdown",
		"sqlite3_os_init", "sqlite3_os_end",
	}},
	{Name: "Prepare", File: "prepare", Prefixes: []string{
		"sqlite3_prepare", "sqlite3_step", "sqlite3_finalize", "sqlite3_reset",
		"sqlite3_sql", "sqlite3_expanded_sql", "sqlite3_normalized_sql",
		"sqlite3_stmt_",
	}},
	{Name: "Column", File: "column", Prefixes: []string{"sqlite3_column_", "sqlite3_data_count"}},
	{Name: "Bind", File: "bind", Prefixes: []string{"sqlite3_bind_", "sqlite3_clear_bindings"}},
	{Name: "Metadata", File: "metadata", Prefixes: []string{
		"sqlite3_last_insert_rowid", "sqlite3_set_last_insert_rowid",
		"sqlite3_changes", "sqlite3_total_changes",
		"sqlite3_errcode", "sqlite3_extended_errcode", "sqlite3_errmsg", "sqlite3_errstr",
		"sqlite3_error_offset",
	}},
	{Name: "Exec", File: "exec", Prefixes: []string{"sqlite3_exec", "sqlite3_interrupt", "sqlite3_is_interrupted"}},
	{Name: "Backup", File: "backup", Prefixes: []string{"sqlite3_backup_"}},
	{Name: "Blob", File: "blob", Prefixes: []string{"sqlite3_blob_"}},
	{Name: "Context", File: "context", Prefixes: []string{"sqlite3_context_", "sqlite3_aggregate_context"}},
	{Name: "Value", File: "value", Prefixes: []string{"sqlite3_value_"}},
	{Name: "Result", File: "result", Prefixes: []string{"sqlite3_result_"}},
	{Name: "VFS", File: "vfs", Prefixes: []string{"sqlite3_vfs_"}},
	{Name: "Wal", File: "wal", Prefixes: []string{"sqlite3_wal_"}},
	{Name: "Serialize", File: "serialize", Prefixes: []string{"sqlite3_serialize", "sqlite3_deserialize"}},
	{Name: "Str", File: "str", Prefixes: []string{"sqlite3_str_"}},
	{Name: "Vtab", File: "vtab", Prefixes: []string{
		"sqlite3_vtab_", "sqlite3_declare_vtab", "sqlite3_overload_function",
	}},
	{Name: "Auth", File: "auth", Prefixes: []string{"sqlite3_set_authorizer"}},
	{Name: "Trace", File: "trace", Prefixes: []string{"sqlite3_trace", "sqlite3_profile"}},
	{Name: "Hook", File: "hook", Prefixes: []string{
		"sqlite3_commit_hook", "sqlite3_rollback_hook", "sqlite3_update_hook",
		"sqlite3_preupdate_", "sqlite3_autovacuum_pages",
	}},
	{Name: "Config", File: "config", Prefixes: []string{
		"sqlite3_config", "sqlite3_db_config", "sqlite3_threadsafe",
		"sqlite3_compileoption_", "sqlite3_extended_result_codes",
	}},
	{Name: "Memory", File: "memory", Prefixes: []string{
		"sqlite3_malloc", "sqlite3_realloc", "sqlite3_free",
		"sqlite3_memory_used", "sqlite3_memory_highwater",
		"sqlite3_soft_heap_limit", "sqlite3_hard_heap_limit",
		"sqlite3_release_memory", "sqlite3_db_release_memory", "sqlite3_msize",
	}},
	{Name: "Func", File: "func", Prefixes: []string{
		"sqlite3_create_function", "sqlite3_create_window_function",
		"sqlite3_create_collation",
	}},
}

// LookupOverride returns the override for the given C function name, or nil.
func LookupOverride(name string) *Override {
	for i := range Overrides {
		if Overrides[i].Name == name {
			return &Overrides[i]
		}
	}
	return nil
}
