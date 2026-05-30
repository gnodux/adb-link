package services

import (
	"testing"

	"github.com/gnodux/adb-link/internal/models"
)

// helper: pointer to string.
func strPtr(s string) *string { return &s }

func newTestMetadataService() *MetadataService {
	return NewMetadataService([]*models.MetadataConfig{
		{
			Datasource: "pg1",
			Databases: map[string]models.DatabaseMeta{
				"mydb": {Comment: "main database"},
			},
			Tables: map[string]models.TableMeta{
				"users": {
					Comment: "user table",
					Columns: map[string]models.ColumnMeta{
						"name":  {Comment: "user name"},
						"email": {Comment: "user email"},
					},
				},
			},
			Views: map[string]models.TableMeta{
				"active_users": {
					Comment: "active users view",
					Columns: map[string]models.ColumnMeta{
						"name": {Comment: "active user name"},
					},
				},
			},
		},
	})
}

// --- UpdateState ---

func TestMetadataService_UpdateState_SingleConfig(t *testing.T) {
	ms := newTestMetadataService()
	meta, ok := ms.lookup("pg1")
	if !ok {
		t.Fatal("pg1 should be present after UpdateState")
	}
	if meta.Databases["mydb"].Comment != "main database" {
		t.Errorf("expected 'main database', got %q", meta.Databases["mydb"].Comment)
	}
}

func TestMetadataService_UpdateState_MergeMultipleConfigs(t *testing.T) {
	ms := NewMetadataService([]*models.MetadataConfig{
		{
			Datasource: "pg1",
			Databases: map[string]models.DatabaseMeta{
				"db1": {Comment: "first database"},
			},
			Tables: map[string]models.TableMeta{
				"t1": {Comment: "table one"},
			},
		},
		{
			Datasource: "pg1",
			Databases: map[string]models.DatabaseMeta{
				"db2": {Comment: "second database"},
			},
			Tables: map[string]models.TableMeta{
				"t2": {Comment: "table two"},
			},
		},
	})
	meta, ok := ms.lookup("pg1")
	if !ok {
		t.Fatal("pg1 should be present after merge")
	}
	if len(meta.Databases) != 2 {
		t.Errorf("expected 2 databases, got %d", len(meta.Databases))
	}
	if meta.Databases["db1"].Comment != "first database" {
		t.Error("db1 comment missing after merge")
	}
	if meta.Databases["db2"].Comment != "second database" {
		t.Error("db2 comment missing after merge")
	}
	if len(meta.Tables) != 2 {
		t.Errorf("expected 2 tables, got %d", len(meta.Tables))
	}
}

func TestMetadataService_UpdateState_SeparateDatasources(t *testing.T) {
	ms := NewMetadataService([]*models.MetadataConfig{
		{
			Datasource: "pg1",
			Databases: map[string]models.DatabaseMeta{
				"db1": {Comment: "pg1 db"},
			},
		},
		{
			Datasource: "mysql1",
			Databases: map[string]models.DatabaseMeta{
				"db2": {Comment: "mysql1 db"},
			},
		},
	})
	if _, ok := ms.lookup("pg1"); !ok {
		t.Error("pg1 should be present")
	}
	if _, ok := ms.lookup("mysql1"); !ok {
		t.Error("mysql1 should be present")
	}
}

func TestMetadataService_UpdateState_ReplacesOld(t *testing.T) {
	ms := newTestMetadataService()
	// Before update: pg1 exists
	if _, ok := ms.lookup("pg1"); !ok {
		t.Fatal("precondition: pg1 should exist")
	}
	// Replace with a new config
	ms.UpdateState([]*models.MetadataConfig{
		{
			Datasource: "pg2",
			Databases:  map[string]models.DatabaseMeta{"newdb": {Comment: "new"}},
		},
	})
	if _, ok := ms.lookup("pg1"); ok {
		t.Error("pg1 should not exist after UpdateState replaces configs")
	}
	if _, ok := ms.lookup("pg2"); !ok {
		t.Error("pg2 should exist after UpdateState")
	}
}

// --- EnhanceDatabaseNames ---

func TestEnhanceDatabaseNames_AddsComments(t *testing.T) {
	ms := newTestMetadataService()
	dbs := []models.ObjectName{
		{Name: "mydb"},
		{Name: "otherdb"},
	}
	result := ms.EnhanceDatabaseNames("pg1", dbs)
	if result[0].Comment != "main database" {
		t.Errorf("expected 'main database', got %q", result[0].Comment)
	}
	if result[1].Comment != "" {
		t.Errorf("otherdb should have no comment, got %q", result[1].Comment)
	}
}

func TestEnhanceDatabaseNames_NoMetadata(t *testing.T) {
	ms := newTestMetadataService()
	dbs := []models.ObjectName{{Name: "db1"}}
	result := ms.EnhanceDatabaseNames("unknown_ds", dbs)
	if result[0].Comment != "" {
		t.Error("unknown datasource should return unchanged")
	}
}

func TestEnhanceDatabaseNames_ExistingCommentNotOverwritten(t *testing.T) {
	ms := newTestMetadataService()
	dbs := []models.ObjectName{
		{Name: "mydb", Comment: "existing comment"},
	}
	result := ms.EnhanceDatabaseNames("pg1", dbs)
	if result[0].Comment != "existing comment" {
		t.Errorf("existing comment should not be overwritten, got %q", result[0].Comment)
	}
}

func TestEnhanceDatabaseNames_EmptyList(t *testing.T) {
	ms := newTestMetadataService()
	result := ms.EnhanceDatabaseNames("pg1", nil)
	if result != nil {
		t.Errorf("nil input should return nil, got %v", result)
	}
}

// --- EnhanceTableInfo ---

func TestEnhanceTableInfo_AddsTableAndColumnComments(t *testing.T) {
	ms := newTestMetadataService()
	ti := &models.TableInfo{
		Name: "users",
		Columns: []models.ColumnInfo{
			{Name: "name", Type: "varchar"},
			{Name: "email", Type: "varchar"},
			{Name: "id", Type: "int"},
		},
	}
	result := ms.EnhanceTableInfo("pg1", ti)
	if result.Comment == nil || *result.Comment != "user table" {
		t.Error("table comment should be 'user table'")
	}
	if result.Columns[0].Comment == nil || *result.Columns[0].Comment != "user name" {
		t.Error("column 'name' comment should be 'user name'")
	}
	if result.Columns[1].Comment == nil || *result.Columns[1].Comment != "user email" {
		t.Error("column 'email' comment should be 'user email'")
	}
	if result.Columns[2].Comment != nil {
		t.Error("column 'id' should have no comment")
	}
}

func TestEnhanceTableInfo_NilInput(t *testing.T) {
	ms := newTestMetadataService()
	result := ms.EnhanceTableInfo("pg1", nil)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

func TestEnhanceTableInfo_NoMetadataForDatasource(t *testing.T) {
	ms := newTestMetadataService()
	ti := &models.TableInfo{Name: "users"}
	result := ms.EnhanceTableInfo("unknown_ds", ti)
	if result.Comment != nil {
		t.Error("unknown datasource should return unchanged TableInfo")
	}
}

func TestEnhanceTableInfo_NoMetadataForTable(t *testing.T) {
	ms := newTestMetadataService()
	ti := &models.TableInfo{Name: "nonexistent_table"}
	result := ms.EnhanceTableInfo("pg1", ti)
	if result.Comment != nil {
		t.Error("table not in metadata should return unchanged")
	}
}

func TestEnhanceTableInfo_ExistingCommentNotOverwritten(t *testing.T) {
	ms := newTestMetadataService()
	existingComment := "my custom comment"
	existingColComment := "my col comment"
	ti := &models.TableInfo{
		Name:    "users",
		Comment: &existingComment,
		Columns: []models.ColumnInfo{
			{Name: "name", Type: "varchar", Comment: &existingColComment},
		},
	}
	result := ms.EnhanceTableInfo("pg1", ti)
	if *result.Comment != "my custom comment" {
		t.Errorf("existing table comment should not be overwritten, got %q", *result.Comment)
	}
	if *result.Columns[0].Comment != "my col comment" {
		t.Errorf("existing column comment should not be overwritten, got %q", *result.Columns[0].Comment)
	}
}

func TestEnhanceTableInfo_EmptyCommentIsOverwritten(t *testing.T) {
	ms := newTestMetadataService()
	empty := ""
	ti := &models.TableInfo{
		Name:    "users",
		Comment: &empty,
		Columns: []models.ColumnInfo{
			{Name: "name", Type: "varchar", Comment: &empty},
		},
	}
	result := ms.EnhanceTableInfo("pg1", ti)
	if result.Comment == nil || *result.Comment != "user table" {
		t.Error("empty table comment should be filled from metadata")
	}
	if result.Columns[0].Comment == nil || *result.Columns[0].Comment != "user name" {
		t.Error("empty column comment should be filled from metadata")
	}
}

// --- EnhanceViewInfo ---

func TestEnhanceViewInfo_AddsViewAndColumnComments(t *testing.T) {
	ms := newTestMetadataService()
	vi := &models.TableInfo{
		Name: "active_users",
		Columns: []models.ColumnInfo{
			{Name: "name", Type: "varchar"},
		},
	}
	result := ms.EnhanceViewInfo("pg1", vi)
	if result.Comment == nil || *result.Comment != "active users view" {
		t.Error("view comment should be 'active users view'")
	}
	if result.Columns[0].Comment == nil || *result.Columns[0].Comment != "active user name" {
		t.Error("column 'name' comment should be 'active user name'")
	}
}

func TestEnhanceViewInfo_NilInput(t *testing.T) {
	ms := newTestMetadataService()
	result := ms.EnhanceViewInfo("pg1", nil)
	if result != nil {
		t.Error("nil input should return nil")
	}
}

func TestEnhanceViewInfo_NoMetadataForDatasource(t *testing.T) {
	ms := newTestMetadataService()
	vi := &models.TableInfo{Name: "active_users"}
	result := ms.EnhanceViewInfo("unknown_ds", vi)
	if result.Comment != nil {
		t.Error("unknown datasource should return unchanged")
	}
}

func TestEnhanceViewInfo_NoMetadataForView(t *testing.T) {
	ms := newTestMetadataService()
	vi := &models.TableInfo{Name: "nonexistent_view"}
	result := ms.EnhanceViewInfo("pg1", vi)
	if result.Comment != nil {
		t.Error("view not in metadata should return unchanged")
	}
}

func TestEnhanceViewInfo_ExistingCommentNotOverwritten(t *testing.T) {
	ms := newTestMetadataService()
	existing := "custom view comment"
	existingCol := "custom col comment"
	vi := &models.TableInfo{
		Name:    "active_users",
		Comment: &existing,
		Columns: []models.ColumnInfo{
			{Name: "name", Type: "varchar", Comment: &existingCol},
		},
	}
	result := ms.EnhanceViewInfo("pg1", vi)
	if *result.Comment != "custom view comment" {
		t.Errorf("existing view comment should not be overwritten, got %q", *result.Comment)
	}
	if *result.Columns[0].Comment != "custom col comment" {
		t.Errorf("existing column comment should not be overwritten, got %q", *result.Columns[0].Comment)
	}
}

func TestEnhanceViewInfo_EmptyCommentIsOverwritten(t *testing.T) {
	ms := newTestMetadataService()
	empty := ""
	vi := &models.TableInfo{
		Name:    "active_users",
		Comment: &empty,
		Columns: []models.ColumnInfo{
			{Name: "name", Type: "varchar", Comment: &empty},
		},
	}
	result := ms.EnhanceViewInfo("pg1", vi)
	if result.Comment == nil || *result.Comment != "active users view" {
		t.Error("empty view comment should be filled from metadata")
	}
	if result.Columns[0].Comment == nil || *result.Columns[0].Comment != "active user name" {
		t.Error("empty column comment should be filled from metadata")
	}
}

// --- Merge with Views ---

func TestMetadataService_UpdateState_MergeViews(t *testing.T) {
	ms := NewMetadataService([]*models.MetadataConfig{
		{
			Datasource: "pg1",
			Views: map[string]models.TableMeta{
				"v1": {Comment: "view one"},
			},
		},
		{
			Datasource: "pg1",
			Views: map[string]models.TableMeta{
				"v2": {Comment: "view two"},
			},
		},
	})
	meta, ok := ms.lookup("pg1")
	if !ok {
		t.Fatal("pg1 should exist")
	}
	if len(meta.Views) != 2 {
		t.Errorf("expected 2 views after merge, got %d", len(meta.Views))
	}
	if meta.Views["v1"].Comment != "view one" {
		t.Error("v1 comment missing")
	}
	if meta.Views["v2"].Comment != "view two" {
		t.Error("v2 comment missing")
	}
}

// --- Empty configs ---

func TestMetadataService_NewWithNilConfigs(t *testing.T) {
	ms := NewMetadataService(nil)
	if _, ok := ms.lookup("anything"); ok {
		t.Error("nil configs should result in empty metadata")
	}
}

func TestMetadataService_NewWithEmptySlice(t *testing.T) {
	ms := NewMetadataService([]*models.MetadataConfig{})
	dbs := []models.ObjectName{{Name: "db1"}}
	result := ms.EnhanceDatabaseNames("pg1", dbs)
	if result[0].Comment != "" {
		t.Error("empty configs should not enhance anything")
	}
}
