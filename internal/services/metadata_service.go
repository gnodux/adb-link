package services

import (
	"sync"

	"github.com/gnodux/adb-link/internal/models"
)

// MetadataService enhances schema results with configured comments.
type MetadataService struct {
	mu       sync.RWMutex
	metadata map[string]*models.MetadataConfig
}

// NewMetadataService creates a new MetadataService.
func NewMetadataService(configs []*models.MetadataConfig) *MetadataService {
	m := &MetadataService{}
	m.UpdateState(configs)
	return m
}

// UpdateState rebuilds the merged metadata index from the supplied configs.
// Safe to call concurrently with readers.
func (m *MetadataService) UpdateState(configs []*models.MetadataConfig) {
	merged := make(map[string]*models.MetadataConfig)
	for _, cfg := range configs {
		if existing, ok := merged[cfg.Datasource]; ok {
			// Merge databases, tables, views
			if existing.Databases == nil {
				existing.Databases = make(map[string]models.DatabaseMeta)
			}
			for k, v := range cfg.Databases {
				existing.Databases[k] = v
			}
			if existing.Tables == nil {
				existing.Tables = make(map[string]models.TableMeta)
			}
			for k, v := range cfg.Tables {
				existing.Tables[k] = v
			}
			if existing.Views == nil {
				existing.Views = make(map[string]models.TableMeta)
			}
			for k, v := range cfg.Views {
				existing.Views[k] = v
			}
		} else {
			merged[cfg.Datasource] = cfg
		}
	}
	m.mu.Lock()
	m.metadata = merged
	m.mu.Unlock()
}

func (m *MetadataService) lookup(datasource string) (*models.MetadataConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	meta, ok := m.metadata[datasource]
	return meta, ok
}

// EnhanceDatabaseNames adds metadata comments to a list of database names.
func (m *MetadataService) EnhanceDatabaseNames(datasource string, dbs []models.ObjectName) []models.ObjectName {
	meta, ok := m.lookup(datasource)
	if !ok {
		return dbs
	}
	for i := range dbs {
		if dbs[i].Comment == "" {
			if dbMeta, ok := meta.Databases[dbs[i].Name]; ok {
				dbs[i].Comment = dbMeta.Comment
			}
		}
	}
	return dbs
}

// EnhanceTableInfo merges metadata comments into a TableInfo (and its columns).
func (m *MetadataService) EnhanceTableInfo(datasource string, ti *models.TableInfo) *models.TableInfo {
	if ti == nil {
		return ti
	}
	meta, ok := m.lookup(datasource)
	if !ok {
		return ti
	}
	tableMeta, ok := meta.Tables[ti.Name]
	if !ok {
		return ti
	}
	if (ti.Comment == nil || *ti.Comment == "") && tableMeta.Comment != "" {
		c := tableMeta.Comment
		ti.Comment = &c
	}
	for i := range ti.Columns {
		col := &ti.Columns[i]
		if col.Comment == nil || *col.Comment == "" {
			if cm, ok := tableMeta.Columns[col.Name]; ok && cm.Comment != "" {
				c := cm.Comment
				col.Comment = &c
			}
		}
	}
	return ti
}

// EnhanceViewInfo merges metadata for views.
func (m *MetadataService) EnhanceViewInfo(datasource string, vi *models.TableInfo) *models.TableInfo {
	if vi == nil {
		return vi
	}
	meta, ok := m.lookup(datasource)
	if !ok {
		return vi
	}
	viewMeta, ok := meta.Views[vi.Name]
	if !ok {
		return vi
	}
	if (vi.Comment == nil || *vi.Comment == "") && viewMeta.Comment != "" {
		c := viewMeta.Comment
		vi.Comment = &c
	}
	for i := range vi.Columns {
		col := &vi.Columns[i]
		if col.Comment == nil || *col.Comment == "" {
			if cm, ok := viewMeta.Columns[col.Name]; ok && cm.Comment != "" {
				c := cm.Comment
				col.Comment = &c
			}
		}
	}
	return vi
}
