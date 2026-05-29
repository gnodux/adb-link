package services

import (
	"context"
	"fmt"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/dialects"
	"github.com/gnodux/adb-link/internal/models"
)

// SchemaService handles database schema introspection.
type SchemaService struct {
	configService     *config.ConfigService
	connectionService *ConnectionService
	metadataService   *MetadataService
	permissionService *PermissionService
}

// NewSchemaService creates a new SchemaService.
func NewSchemaService(
	cs *config.ConfigService,
	conn *ConnectionService,
	meta *MetadataService,
	perm *PermissionService,
) *SchemaService {
	return &SchemaService{
		configService:     cs,
		connectionService: conn,
		metadataService:   meta,
		permissionService: perm,
	}
}

// GetDatabases lists all databases in a datasource.
func (s *SchemaService) GetDatabases(ctx context.Context, datasourceName, userName string) ([]models.ObjectName, error) {
	if !s.permissionService.CheckDatasource(userName, datasourceName) {
		return nil, fmt.Errorf("access denied: user '%s' cannot access datasource '%s'", userName, datasourceName)
	}
	cfg, err := s.configService.GetDatasource(datasourceName)
	if err != nil {
		return nil, err
	}

	var databases []models.ObjectName
	if IsNonSQLType(cfg.Type) {
		client, _, err := s.connectionService.GetNonSQLClient(datasourceName)
		if err != nil {
			return nil, err
		}
		databases, err = client.GetDatabases(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		db, _, err := s.connectionService.GetSQLDB(datasourceName, "")
		if err != nil {
			return nil, err
		}
		dialect, err := dialects.GetDialect(cfg.Type)
		if err != nil {
			return nil, err
		}
		databases, err = dialect.GetDatabases(ctx, db)
		if err != nil {
			return nil, err
		}
	}

	databases = s.metadataService.EnhanceDatabaseNames(datasourceName, databases)
	// Filter by permission
	names := make([]string, len(databases))
	for i, d := range databases {
		names[i] = d.Name
	}
	allowed := s.permissionService.FilterDatabases(userName, datasourceName, names)
	allowedSet := make(map[string]bool)
	for _, n := range allowed {
		allowedSet[n] = true
	}
	var result []models.ObjectName
	for _, d := range databases {
		if allowedSet[d.Name] {
			result = append(result, d)
		}
	}
	return result, nil
}

// GetSchema returns the full schema (tables + views) for a database.
func (s *SchemaService) GetSchema(ctx context.Context, datasourceName, database, userName string) (*models.DatabaseSchema, error) {
	cfg, err := s.configService.GetDatasource(datasourceName)
	if err != nil {
		return nil, err
	}

	var tables, views []models.TableInfo

	if IsNonSQLType(cfg.Type) {
		client, _, err := s.connectionService.GetNonSQLClient(datasourceName)
		if err != nil {
			return nil, err
		}
		tableNames, err := client.GetTableNames(ctx, database)
		if err != nil {
			return nil, err
		}
		for _, t := range tableNames {
			ti, err := client.GetTableInfo(ctx, database, t.Name)
			if err != nil {
				return nil, err
			}
			tables = append(tables, *ti)
		}
	} else {
		db, _, err := s.connectionService.GetSQLDB(datasourceName, database)
		if err != nil {
			return nil, err
		}
		dialect, err := dialects.GetDialect(cfg.Type)
		if err != nil {
			return nil, err
		}
		tableNames, err := dialect.GetTableNames(ctx, db, database)
		if err != nil {
			return nil, err
		}
		for _, t := range tableNames {
			ti, err := dialect.GetTableInfo(ctx, db, database, t.Name)
			if err != nil {
				return nil, err
			}
			tables = append(tables, *ti)
		}
		viewNames, err := dialect.GetViewNames(ctx, db, database)
		if err == nil {
			for _, v := range viewNames {
				vi, err := dialect.GetViewInfo(ctx, db, database, v.Name)
				if err == nil {
					views = append(views, *vi)
				}
			}
		}
	}

	// Enhance with metadata
	for i := range tables {
		s.metadataService.EnhanceTableInfo(datasourceName, &tables[i])
	}
	for i := range views {
		s.metadataService.EnhanceViewInfo(datasourceName, &views[i])
	}

	// Filter by permission
	tableNames := make([]string, len(tables))
	for i, t := range tables {
		tableNames[i] = t.Name
	}
	allowedTables := s.permissionService.FilterTables(userName, datasourceName, database, tableNames)
	allowedSet := make(map[string]bool)
	for _, n := range allowedTables {
		allowedSet[n] = true
	}
	var filteredTables []models.TableInfo
	for _, t := range tables {
		if allowedSet[t.Name] {
			filteredTables = append(filteredTables, t)
		}
	}

	viewNames := make([]string, len(views))
	for i, v := range views {
		viewNames[i] = v.Name
	}
	allowedViews := s.permissionService.FilterTables(userName, datasourceName, database, viewNames)
	allowedViewSet := make(map[string]bool)
	for _, n := range allowedViews {
		allowedViewSet[n] = true
	}
	var filteredViews []models.TableInfo
	for _, v := range views {
		if allowedViewSet[v.Name] {
			filteredViews = append(filteredViews, v)
		}
	}

	return &models.DatabaseSchema{
		DatabaseName: database,
		Tables:       filteredTables,
		Views:        filteredViews,
	}, nil
}

// GetTableInfo returns column info for a single table.
func (s *SchemaService) GetTableInfo(ctx context.Context, datasourceName, database, table, userName string) (*models.TableInfo, error) {
	if !s.permissionService.CheckTable(userName, datasourceName, database, table) {
		return nil, fmt.Errorf("access denied: user '%s' cannot access table '%s/%s/%s'", userName, datasourceName, database, table)
	}
	cfg, err := s.configService.GetDatasource(datasourceName)
	if err != nil {
		return nil, err
	}

	var ti *models.TableInfo
	if IsNonSQLType(cfg.Type) {
		client, _, err := s.connectionService.GetNonSQLClient(datasourceName)
		if err != nil {
			return nil, err
		}
		ti, err = client.GetTableInfo(ctx, database, table)
		if err != nil {
			return nil, err
		}
	} else {
		db, _, err := s.connectionService.GetSQLDB(datasourceName, database)
		if err != nil {
			return nil, err
		}
		dialect, err := dialects.GetDialect(cfg.Type)
		if err != nil {
			return nil, err
		}
		ti, err = dialect.GetTableInfo(ctx, db, database, table)
		if err != nil {
			return nil, err
		}
	}

	ti = s.metadataService.EnhanceTableInfo(datasourceName, ti)
	// Filter columns
	colNames := make([]string, len(ti.Columns))
	for i, c := range ti.Columns {
		colNames[i] = c.Name
	}
	allowed := s.permissionService.FilterFields(userName, datasourceName, database, table, colNames)
	allowedSet := make(map[string]bool)
	for _, n := range allowed {
		allowedSet[n] = true
	}
	var filtered []models.ColumnInfo
	for _, c := range ti.Columns {
		if allowedSet[c.Name] {
			filtered = append(filtered, c)
		}
	}
	ti.Columns = filtered
	return ti, nil
}

// GetViewInfo returns column info for a single view.
func (s *SchemaService) GetViewInfo(ctx context.Context, datasourceName, database, view, userName string) (*models.TableInfo, error) {
	if !s.permissionService.CheckTable(userName, datasourceName, database, view) {
		return nil, fmt.Errorf("access denied: user '%s' cannot access view '%s/%s/%s'", userName, datasourceName, database, view)
	}
	cfg, err := s.configService.GetDatasource(datasourceName)
	if err != nil {
		return nil, err
	}

	var vi *models.TableInfo
	if IsNonSQLType(cfg.Type) {
		// Non-SQL databases typically don't have views
		vi = &models.TableInfo{Name: view}
	} else {
		db, _, err := s.connectionService.GetSQLDB(datasourceName, database)
		if err != nil {
			return nil, err
		}
		dialect, err := dialects.GetDialect(cfg.Type)
		if err != nil {
			return nil, err
		}
		vi, err = dialect.GetViewInfo(ctx, db, database, view)
		if err != nil {
			return nil, err
		}
	}

	vi = s.metadataService.EnhanceViewInfo(datasourceName, vi)
	colNames := make([]string, len(vi.Columns))
	for i, c := range vi.Columns {
		colNames[i] = c.Name
	}
	allowed := s.permissionService.FilterFields(userName, datasourceName, database, view, colNames)
	allowedSet := make(map[string]bool)
	for _, n := range allowed {
		allowedSet[n] = true
	}
	var filtered []models.ColumnInfo
	for _, c := range vi.Columns {
		if allowedSet[c.Name] {
			filtered = append(filtered, c)
		}
	}
	vi.Columns = filtered
	return vi, nil
}
