package services

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/gnodux/adb-link/internal/models"
)

// PermissionService checks user permissions on datasources, databases, tables, fields and tools.
// Default policy: deny all unless explicitly granted.
// Exception: when no user is associated with a request (empty userName, e.g.
// stdio MCP transport with no auth configured), all checks are bypassed. Once
// any auth user is configured, every named user — including "mcp_user" — must
// be granted access through explicit permission rules.
//
// State is held behind an RWMutex so that ConfigService hot-reload can swap
// the rule set without race conditions.
type PermissionService struct {
	mu              sync.RWMutex
	authUsersByName map[string]*models.AuthUser
	permissions     []*models.PermissionConfig
}

// NewPermissionService creates a new PermissionService.
func NewPermissionService(authUsers map[string]*models.AuthUser, permissions []*models.PermissionConfig) *PermissionService {
	ps := &PermissionService{}
	ps.UpdateState(authUsers, permissions)
	return ps
}

// UpdateState replaces the auth-user index and permission rule list. Safe to
// call concurrently with permission checks.
func (ps *PermissionService) UpdateState(authUsers map[string]*models.AuthUser, permissions []*models.PermissionConfig) {
	byName := make(map[string]*models.AuthUser, len(authUsers))
	for _, u := range authUsers {
		byName[u.Name] = u
	}
	perms := append([]*models.PermissionConfig(nil), permissions...)
	ps.mu.Lock()
	ps.authUsersByName = byName
	ps.permissions = perms
	ps.mu.Unlock()
}

func (ps *PermissionService) getUserGroup(userName string) string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if u, ok := ps.authUsersByName[userName]; ok {
		return u.Group
	}
	return ""
}

func (ps *PermissionService) getUserPermissions(userName string) []*models.PermissionConfig {
	group := ps.getUserGroup(userName)
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	var matched []*models.PermissionConfig
	for _, perm := range ps.permissions {
		if containsString(perm.Users, userName) {
			matched = append(matched, perm)
			continue
		}
		if group != "" && containsString(perm.Groups, group) {
			matched = append(matched, perm)
		}
	}
	return matched
}

func (ps *PermissionService) collectRules(userName string) []models.PermissionRule {
	perms := ps.getUserPermissions(userName)
	var rules []models.PermissionRule
	for _, p := range perms {
		rules = append(rules, p.Rules...)
	}
	return rules
}

func (ps *PermissionService) collectTools(userName string) []string {
	perms := ps.getUserPermissions(userName)
	var tools []string
	for _, p := range perms {
		tools = append(tools, p.Tools...)
	}
	return tools
}

// isBypassUser returns true only when no user identity is attached to the
// request — typically because authentication is not configured. Named users
// (including "mcp_user") never bypass permission checks.
func (ps *PermissionService) isBypassUser(userName string) bool {
	return userName == ""
}

// CheckDatasource verifies if a user can access a datasource.
func (ps *PermissionService) CheckDatasource(userName, datasourceName string) bool {
	if ps.isBypassUser(userName) {
		return true
	}
	rules := ps.collectRules(userName)
	if len(rules) == 0 {
		return false
	}
	for _, r := range rules {
		if matchPattern(datasourceName, r.Datasource) {
			return true
		}
	}
	return false
}

// CheckDatabase verifies database access.
func (ps *PermissionService) CheckDatabase(userName, datasourceName, database string) bool {
	if ps.isBypassUser(userName) {
		return true
	}
	for _, r := range ps.collectRules(userName) {
		if matchPattern(datasourceName, r.Datasource) && matchAny(database, r.Databases) {
			return true
		}
	}
	return false
}

// CheckTable verifies table access.
func (ps *PermissionService) CheckTable(userName, datasourceName, database, table string) bool {
	if ps.isBypassUser(userName) {
		return true
	}
	for _, r := range ps.collectRules(userName) {
		if matchPattern(datasourceName, r.Datasource) && matchAny(database, r.Databases) && matchAny(table, r.Tables) {
			return true
		}
	}
	return false
}

// CheckField verifies field access.
func (ps *PermissionService) CheckField(userName, datasourceName, database, table, field string) bool {
	if ps.isBypassUser(userName) {
		return true
	}
	for _, r := range ps.collectRules(userName) {
		if matchPattern(datasourceName, r.Datasource) && matchAny(database, r.Databases) && matchAny(table, r.Tables) && matchAny(field, r.Fields) {
			return true
		}
	}
	return false
}

// CheckTool verifies tool execution.
func (ps *PermissionService) CheckTool(userName, toolName string) bool {
	if ps.isBypassUser(userName) {
		return true
	}
	patterns := ps.collectTools(userName)
	if len(patterns) == 0 {
		return false
	}
	for _, p := range patterns {
		if matchPattern(toolName, p) {
			return true
		}
	}
	return false
}

// FilterDatabases returns only databases accessible to the user.
func (ps *PermissionService) FilterDatabases(userName, datasourceName string, databases []string) []string {
	if ps.isBypassUser(userName) {
		return databases
	}
	var out []string
	for _, db := range databases {
		if ps.CheckDatabase(userName, datasourceName, db) {
			out = append(out, db)
		}
	}
	return out
}

// FilterTables returns only tables accessible to the user.
func (ps *PermissionService) FilterTables(userName, datasourceName, database string, tables []string) []string {
	if ps.isBypassUser(userName) {
		return tables
	}
	var out []string
	for _, t := range tables {
		if ps.CheckTable(userName, datasourceName, database, t) {
			out = append(out, t)
		}
	}
	return out
}

// FilterFields returns only fields accessible to the user.
func (ps *PermissionService) FilterFields(userName, datasourceName, database, table string, fields []string) []string {
	if ps.isBypassUser(userName) {
		return fields
	}
	var out []string
	for _, f := range fields {
		if ps.CheckField(userName, datasourceName, database, table, f) {
			out = append(out, f)
		}
	}
	return out
}

// FilterTools returns only tools the user can execute.
func (ps *PermissionService) FilterTools(userName string, toolNames []string) []string {
	if ps.isBypassUser(userName) {
		return toolNames
	}
	var out []string
	for _, t := range toolNames {
		if ps.CheckTool(userName, t) {
			out = append(out, t)
		}
	}
	return out
}

// matchPattern checks if a name matches a glob pattern (using filepath.Match semantics).
func matchPattern(name, pattern string) bool {
	if pattern == "*" || pattern == name {
		return true
	}
	matched, err := filepath.Match(pattern, name)
	if err != nil {
		return false
	}
	return matched
}

func matchAny(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, p := range patterns {
		if matchPattern(name, p) {
			return true
		}
	}
	return false
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// _ keeps strings import for future use
var _ = strings.HasPrefix
