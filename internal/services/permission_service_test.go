package services

import (
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

// helper to build a PermissionService with a standard test fixture.
func newTestPermissionService() *PermissionService {
	return NewPermissionService(
		map[string]*models.AuthUser{
			"alice": {Name: "alice", Group: "analysts"},
			"bob":   {Name: "bob", Group: "engineers"},
			"carol": {Name: "carol"},
		},
		[]*models.PermissionConfig{
			{
				Users: []string{"alice"},
				Rules: []models.PermissionRule{
					{Datasource: "prod_*", Databases: []string{"analytics"}, Tables: []string{"*"}, Fields: []string{"*"}},
				},
				Tools: []string{"query_*"},
			},
			{
				Groups: []string{"analysts"},
				Rules: []models.PermissionRule{
					{Datasource: "staging", Databases: []string{"*"}, Tables: []string{"user_*"}, Fields: []string{"*"}},
				},
				Tools: []string{"list_*"},
			},
			{
				Users: []string{"bob"},
				Rules: []models.PermissionRule{
					{Datasource: "prod_main", Databases: []string{"main"}, Tables: []string{"orders"}, Fields: []string{"id", "total"}},
				},
				Tools: []string{"query_sql"},
			},
		},
	)
}

// --- Bypass user tests ---

func TestBypassUser_CheckDatasource(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckDatasource("", "anything") {
		t.Error("bypass user should have access to any datasource")
	}
}

func TestBypassUser_CheckDatabase(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckDatabase("", "ds", "db") {
		t.Error("bypass user should have access to any database")
	}
}

func TestBypassUser_CheckTable(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckTable("", "ds", "db", "tbl") {
		t.Error("bypass user should have access to any table")
	}
}

func TestBypassUser_CheckField(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckField("", "ds", "db", "tbl", "col") {
		t.Error("bypass user should have access to any field")
	}
}

func TestBypassUser_CheckTool(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckTool("", "any_tool") {
		t.Error("bypass user should have access to any tool")
	}
}

func TestBypassUser_FilterDatabases(t *testing.T) {
	ps := newTestPermissionService()
	input := []string{"db1", "db2"}
	got := ps.FilterDatabases("", "ds", input)
	if len(got) != len(input) {
		t.Errorf("bypass user FilterDatabases: got %v, want %v", got, input)
	}
}

func TestBypassUser_FilterTables(t *testing.T) {
	ps := newTestPermissionService()
	input := []string{"t1", "t2"}
	got := ps.FilterTables("", "ds", "db", input)
	if len(got) != len(input) {
		t.Errorf("bypass user FilterTables: got %v, want %v", got, input)
	}
}

func TestBypassUser_FilterFields(t *testing.T) {
	ps := newTestPermissionService()
	input := []string{"f1", "f2"}
	got := ps.FilterFields("", "ds", "db", "tbl", input)
	if len(got) != len(input) {
		t.Errorf("bypass user FilterFields: got %v, want %v", got, input)
	}
}

func TestBypassUser_FilterTools(t *testing.T) {
	ps := newTestPermissionService()
	input := []string{"tool1", "tool2"}
	got := ps.FilterTools("", input)
	if len(got) != len(input) {
		t.Errorf("bypass user FilterTools: got %v, want %v", got, input)
	}
}

// --- No rules tests (user exists but has no permissions) ---

func TestNoRules_CheckDatasource(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckDatasource("carol", "prod_main") {
		t.Error("user with no rules should not access any datasource")
	}
}

func TestNoRules_CheckDatabase(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckDatabase("carol", "prod_main", "main") {
		t.Error("user with no rules should not access any database")
	}
}

func TestNoRules_CheckTable(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckTable("carol", "prod_main", "main", "orders") {
		t.Error("user with no rules should not access any table")
	}
}

func TestNoRules_CheckField(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckField("carol", "prod_main", "main", "orders", "id") {
		t.Error("user with no rules should not access any field")
	}
}

func TestNoRules_CheckTool(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckTool("carol", "query_sql") {
		t.Error("user with no rules should not access any tool")
	}
}

// --- Exact match tests ---

func TestExactMatch_CheckDatasource(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckDatasource("bob", "prod_main") {
		t.Error("bob should access prod_main (exact match)")
	}
}

func TestExactMatch_CheckDatabase(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckDatabase("bob", "prod_main", "main") {
		t.Error("bob should access prod_main.main")
	}
}

func TestExactMatch_CheckTable(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckTable("bob", "prod_main", "main", "orders") {
		t.Error("bob should access prod_main.main.orders")
	}
}

func TestExactMatch_CheckField(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckField("bob", "prod_main", "main", "orders", "id") {
		t.Error("bob should access prod_main.main.orders.id")
	}
	if !ps.CheckField("bob", "prod_main", "main", "orders", "total") {
		t.Error("bob should access prod_main.main.orders.total")
	}
}

func TestExactMatch_CheckTool(t *testing.T) {
	ps := newTestPermissionService()
	if !ps.CheckTool("bob", "query_sql") {
		t.Error("bob should access query_sql tool")
	}
}

// --- Glob matching tests ---

func TestGlobMatching_DatasourcePattern(t *testing.T) {
	ps := newTestPermissionService()
	// alice has rule with Datasource "prod_*"
	if !ps.CheckDatasource("alice", "prod_main") {
		t.Error("alice should match prod_main via prod_*")
	}
	if !ps.CheckDatasource("alice", "prod_secondary") {
		t.Error("alice should match prod_secondary via prod_*")
	}
	if ps.CheckDatasource("alice", "staging") {
		// alice does not have a direct "staging" user rule, but she is in group analysts
		// which grants "staging" -- so this should actually be true
	}
}

func TestGlobMatching_DatabaseWildcard(t *testing.T) {
	ps := newTestPermissionService()
	// alice via group analysts gets staging with Databases: ["*"]
	if !ps.CheckDatabase("alice", "staging", "any_db") {
		t.Error("alice should access any database on staging via group analysts")
	}
}

func TestGlobMatching_TablePattern(t *testing.T) {
	ps := newTestPermissionService()
	// alice via group analysts gets staging.* with Tables: ["user_*"]
	if !ps.CheckTable("alice", "staging", "mydb", "user_profiles") {
		t.Error("alice should access user_profiles via user_*")
	}
	if ps.CheckTable("alice", "staging", "mydb", "order_items") {
		t.Error("alice should not access order_items via user_*")
	}
}

func TestGlobMatching_FieldWildcard(t *testing.T) {
	ps := newTestPermissionService()
	// alice via group analysts gets staging.*.user_*.["*"]
	if !ps.CheckField("alice", "staging", "mydb", "user_profiles", "email") {
		t.Error("alice should access any field on user_profiles")
	}
}

func TestGlobMatching_ToolPattern(t *testing.T) {
	ps := newTestPermissionService()
	// alice has Tools: ["query_*"] directly
	if !ps.CheckTool("alice", "query_sql") {
		t.Error("alice should access query_sql via query_*")
	}
	if !ps.CheckTool("alice", "query_fulltext") {
		t.Error("alice should access query_fulltext via query_*")
	}
	// alice via group analysts gets Tools: ["list_*"]
	if !ps.CheckTool("alice", "list_tables") {
		t.Error("alice should access list_tables via list_* (group)")
	}
	if ps.CheckTool("alice", "delete_all") {
		t.Error("alice should not access delete_all")
	}
}

// --- No match tests ---

func TestNoMatch_CheckDatasource(t *testing.T) {
	ps := newTestPermissionService()
	// bob only has prod_main, not staging
	if ps.CheckDatasource("bob", "staging") {
		t.Error("bob should not access staging")
	}
}

func TestNoMatch_CheckDatabase(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckDatabase("bob", "prod_main", "other_db") {
		t.Error("bob should not access prod_main.other_db")
	}
}

func TestNoMatch_CheckTable(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckTable("bob", "prod_main", "main", "users") {
		t.Error("bob should not access prod_main.main.users")
	}
}

func TestNoMatch_CheckField(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckField("bob", "prod_main", "main", "orders", "secret") {
		t.Error("bob should not access prod_main.main.orders.secret")
	}
}

func TestNoMatch_CheckTool(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckTool("bob", "delete_all") {
		t.Error("bob should not access delete_all tool")
	}
}

// --- Filter methods ---

func TestFilterDatabases(t *testing.T) {
	ps := newTestPermissionService()
	// bob has access to prod_main.main only
	got := ps.FilterDatabases("bob", "prod_main", []string{"main", "other", "test"})
	if len(got) != 1 || got[0] != "main" {
		t.Errorf("FilterDatabases for bob: got %v, want [main]", got)
	}
}

func TestFilterDatabases_Bypass(t *testing.T) {
	ps := newTestPermissionService()
	input := []string{"a", "b", "c"}
	got := ps.FilterDatabases("", "ds", input)
	if len(got) != 3 {
		t.Errorf("FilterDatabases bypass: got %v, want %v", got, input)
	}
}

func TestFilterTables(t *testing.T) {
	ps := newTestPermissionService()
	// alice via group analysts on staging has Tables: ["user_*"]
	got := ps.FilterTables("alice", "staging", "mydb", []string{"user_profiles", "orders", "user_settings"})
	if len(got) != 2 {
		t.Errorf("FilterTables for alice on staging: got %v, want [user_profiles user_settings]", got)
	}
}

func TestFilterFields(t *testing.T) {
	ps := newTestPermissionService()
	// bob has access to fields id, total on prod_main.main.orders
	got := ps.FilterFields("bob", "prod_main", "main", "orders", []string{"id", "secret", "total", "password"})
	if len(got) != 2 {
		t.Errorf("FilterFields for bob: got %v, want [id total]", got)
	}
}

func TestFilterTools(t *testing.T) {
	ps := newTestPermissionService()
	// bob has Tools: ["query_sql"]
	got := ps.FilterTools("bob", []string{"query_sql", "delete_all", "list_tables"})
	if len(got) != 1 || got[0] != "query_sql" {
		t.Errorf("FilterTools for bob: got %v, want [query_sql]", got)
	}
}

func TestFilterTools_AliceMultipleSources(t *testing.T) {
	ps := newTestPermissionService()
	// alice gets query_* from user rule, list_* from group rule
	got := ps.FilterTools("alice", []string{"query_sql", "list_tables", "delete_all"})
	if len(got) != 2 {
		t.Errorf("FilterTools for alice: got %v, want [query_sql list_tables]", got)
	}
}

// --- UpdateState ---

func TestUpdateState_ReplacesRules(t *testing.T) {
	ps := newTestPermissionService()

	// Before update, alice can access prod_main
	if !ps.CheckDatasource("alice", "prod_main") {
		t.Fatal("precondition: alice should access prod_main before update")
	}

	// Replace all rules: only bob has access now
	ps.UpdateState(
		map[string]*models.AuthUser{
			"alice": {Name: "alice", Group: "analysts"},
			"bob":   {Name: "bob"},
		},
		[]*models.PermissionConfig{
			{
				Users: []string{"bob"},
				Rules: []models.PermissionRule{
					{Datasource: "new_ds", Databases: []string{"newdb"}},
				},
			},
		},
	)

	// After update, alice should NOT access prod_main anymore
	if ps.CheckDatasource("alice", "prod_main") {
		t.Error("after UpdateState, alice should not access prod_main")
	}
	// bob should access new_ds
	if !ps.CheckDatasource("bob", "new_ds") {
		t.Error("after UpdateState, bob should access new_ds")
	}
	// bob should NOT access prod_main anymore
	if ps.CheckDatasource("bob", "prod_main") {
		t.Error("after UpdateState, bob should not access prod_main")
	}
}

// --- Group-based permissions ---

func TestGroupPermission(t *testing.T) {
	ps := newTestPermissionService()
	// alice is in group "analysts", which grants staging with Tables: user_*
	if !ps.CheckDatasource("alice", "staging") {
		t.Error("alice should access staging via group analysts")
	}
	if !ps.CheckTable("alice", "staging", "anydb", "user_data") {
		t.Error("alice should access user_data via group analysts")
	}
}

func TestGroupPermission_NoMatchForOtherGroup(t *testing.T) {
	ps := newTestPermissionService()
	// bob is in group "engineers", no permission config targets engineers
	if ps.CheckDatasource("bob", "staging") {
		t.Error("bob should not access staging (engineers group has no permission)")
	}
}

// --- Multiple permissions merged ---

func TestMultiplePermissions_Merged(t *testing.T) {
	ps := newTestPermissionService()
	// alice matches both user-based (prod_*) and group-based (staging) configs
	// Should access both
	if !ps.CheckDatasource("alice", "prod_main") {
		t.Error("alice should access prod_main via user rule")
	}
	if !ps.CheckDatasource("alice", "staging") {
		t.Error("alice should access staging via group rule")
	}
}

// --- matchPattern helper ---

func TestMatchPattern_Exact(t *testing.T) {
	if !matchPattern("hello", "hello") {
		t.Error("exact match should succeed")
	}
}

func TestMatchPattern_Star(t *testing.T) {
	if !matchPattern("anything", "*") {
		t.Error("* should match anything")
	}
}

func TestMatchPattern_Glob(t *testing.T) {
	if !matchPattern("prod_main", "prod_*") {
		t.Error("prod_main should match prod_*")
	}
	if matchPattern("staging", "prod_*") {
		t.Error("staging should not match prod_*")
	}
}

func TestMatchPattern_InvalidGlob(t *testing.T) {
	// Invalid glob pattern with unmatched '[' should return false
	if matchPattern("test", "[invalid") {
		t.Error("invalid glob pattern should return false")
	}
}

func TestMatchPattern_Empty(t *testing.T) {
	if matchPattern("", "something") {
		t.Error("empty name should not match non-empty, non-star pattern")
	}
	if !matchPattern("", "") {
		t.Error("empty name should match empty pattern (exact match)")
	}
}

// --- matchAny helper ---

func TestMatchAny_EmptyPatterns(t *testing.T) {
	if matchAny("something", nil) {
		t.Error("matchAny with nil patterns should return false")
	}
	if matchAny("something", []string{}) {
		t.Error("matchAny with empty patterns should return false")
	}
}

func TestMatchAny_Match(t *testing.T) {
	if !matchAny("hello", []string{"world", "hello"}) {
		t.Error("matchAny should find exact match")
	}
	if !matchAny("prod_main", []string{"dev_*", "prod_*"}) {
		t.Error("matchAny should find glob match")
	}
}

func TestMatchAny_NoMatch(t *testing.T) {
	if matchAny("staging", []string{"prod_*", "dev_*"}) {
		t.Error("matchAny should return false when nothing matches")
	}
}

// --- containsString helper (tested indirectly) ---

func TestContainsString(t *testing.T) {
	if !containsString([]string{"a", "b", "c"}, "b") {
		t.Error("containsString should find 'b'")
	}
	if containsString([]string{"a", "b"}, "z") {
		t.Error("containsString should not find 'z'")
	}
	if containsString(nil, "a") {
		t.Error("containsString on nil slice should return false")
	}
}

// --- Unknown user (not in authUsersByName) ---

func TestUnknownUser_CheckDatasource(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckDatasource("unknown_user", "prod_main") {
		t.Error("unknown user should not access any datasource")
	}
}

func TestUnknownUser_CheckTool(t *testing.T) {
	ps := newTestPermissionService()
	if ps.CheckTool("unknown_user", "query_sql") {
		t.Error("unknown user should not access any tool")
	}
}

// --- Edge cases ---

func TestFilterDatabases_EmptyInput(t *testing.T) {
	ps := newTestPermissionService()
	got := ps.FilterDatabases("alice", "prod_main", nil)
	if got != nil {
		t.Errorf("FilterDatabases with nil input: got %v, want nil", got)
	}
}

func TestFilterTables_EmptyInput(t *testing.T) {
	ps := newTestPermissionService()
	got := ps.FilterTables("alice", "staging", "db", nil)
	if got != nil {
		t.Errorf("FilterTables with nil input: got %v, want nil", got)
	}
}

func TestFilterFields_EmptyInput(t *testing.T) {
	ps := newTestPermissionService()
	got := ps.FilterFields("bob", "prod_main", "main", "orders", nil)
	if got != nil {
		t.Errorf("FilterFields with nil input: got %v, want nil", got)
	}
}

func TestFilterTools_EmptyInput(t *testing.T) {
	ps := newTestPermissionService()
	got := ps.FilterTools("bob", nil)
	if got != nil {
		t.Errorf("FilterTools with nil input: got %v, want nil", got)
	}
}

func TestCheckField_NoFieldPatterns(t *testing.T) {
	// bob's rule has specific fields, so a field not in the list should fail
	ps := newTestPermissionService()
	if ps.CheckField("bob", "prod_main", "main", "orders", "nonexistent") {
		t.Error("bob should not access nonexistent field")
	}
}

func TestCheckDatabase_NoDatabaseInRule(t *testing.T) {
	// Create a service with a rule that has no Databases specified
	ps := NewPermissionService(
		map[string]*models.AuthUser{
			"dave": {Name: "dave"},
		},
		[]*models.PermissionConfig{
			{
				Users: []string{"dave"},
				Rules: []models.PermissionRule{
					{Datasource: "ds1"}, // no Databases
				},
			},
		},
	)
	// CheckDatasource should succeed
	if !ps.CheckDatasource("dave", "ds1") {
		t.Error("dave should access ds1")
	}
	// CheckDatabase should fail because Databases is empty (matchAny returns false)
	if ps.CheckDatabase("dave", "ds1", "anydb") {
		t.Error("dave should not access any database when Databases is empty in rule")
	}
}
