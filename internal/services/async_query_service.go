package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/gnodux/adb-link/internal/config"
	"github.com/gnodux/adb-link/internal/models"
)

// asyncQueryEntry holds the state for one async query execution.
type asyncQueryEntry struct {
	mu          sync.Mutex
	queryID     string
	request     *models.AsyncQueryRequest
	toolConfig  *models.ToolConfig
	toolParams  map[string]any
	status      models.QueryStatus
	createdAt   time.Time
	startedAt   *time.Time
	completedAt *time.Time
	result      *models.QueryResult
	errMessage  string
	userName    string
	cancel      context.CancelFunc
}

// AsyncQueryService manages background query and tool execution tasks.
type AsyncQueryService struct {
	mu             sync.RWMutex
	queryService   *QueryService
	configService  *config.ConfigService
	ttl            time.Duration
	queries        map[string]*asyncQueryEntry
	cleanupCancel  context.CancelFunc
	cleanupRunning bool
}

// NewAsyncQueryService creates a new AsyncQueryService.
func NewAsyncQueryService(qs *QueryService, cs *config.ConfigService, ttlSeconds int) *AsyncQueryService {
	if ttlSeconds <= 0 {
		ttlSeconds = 3600
	}
	return &AsyncQueryService{
		queryService:  qs,
		configService: cs,
		ttl:           time.Duration(ttlSeconds) * time.Second,
		queries:       make(map[string]*asyncQueryEntry),
	}
}

// Start begins the periodic cleanup loop.
func (a *AsyncQueryService) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cleanupRunning {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cleanupCancel = cancel
	a.cleanupRunning = true
	go a.cleanupLoop(ctx)
}

// Stop cancels the cleanup loop and all running queries.
func (a *AsyncQueryService) Stop() {
	a.mu.Lock()
	if a.cleanupCancel != nil {
		a.cleanupCancel()
		a.cleanupCancel = nil
	}
	a.cleanupRunning = false
	queries := make([]*asyncQueryEntry, 0, len(a.queries))
	for _, q := range a.queries {
		queries = append(queries, q)
	}
	a.mu.Unlock()

	for _, q := range queries {
		q.mu.Lock()
		if q.cancel != nil {
			q.cancel()
		}
		q.mu.Unlock()
	}
}

func (a *AsyncQueryService) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.cleanupExpired()
		}
	}
}

func (a *AsyncQueryService) cleanupExpired() {
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, entry := range a.queries {
		if now.Sub(entry.createdAt) > a.ttl {
			entry.mu.Lock()
			if entry.cancel != nil {
				entry.cancel()
			}
			entry.mu.Unlock()
			delete(a.queries, id)
		}
	}
}

// Submit schedules a query for background execution and returns a query ID.
func (a *AsyncQueryService) Submit(req *models.AsyncQueryRequest, userName string) (string, error) {
	queryID := uuid.NewString()
	now := time.Now()
	ctx, cancel := context.WithCancel(context.Background())

	entry := &asyncQueryEntry{
		queryID:   queryID,
		request:   req,
		status:    models.QueryStatusPending,
		createdAt: now,
		userName:  userName,
		cancel:    cancel,
	}

	a.mu.Lock()
	a.queries[queryID] = entry
	a.mu.Unlock()

	AuditLog().Printf("user=%s | action=submit_async_query | query_id=%s | datasource=%s | database=%s | sql=%s",
		userName, queryID, req.DatasourceName, req.Database, truncate(req.SQL, 200))

	go a.runQuery(ctx, entry)
	return queryID, nil
}

func (a *AsyncQueryService) runQuery(ctx context.Context, entry *asyncQueryEntry) {
	entry.mu.Lock()
	entry.status = models.QueryStatusRunning
	startedAt := time.Now()
	entry.startedAt = &startedAt
	req := entry.request
	userName := entry.userName
	entry.mu.Unlock()

	queryReq := &models.QueryRequest{
		DatasourceName: req.DatasourceName,
		Database:       req.Database,
		SQL:            req.SQL,
		Limit:          req.Limit,
		TimeoutSeconds: req.TimeoutSeconds,
	}
	result, err := a.queryService.Execute(ctx, queryReq, userName)

	entry.mu.Lock()
	completedAt := time.Now()
	entry.completedAt = &completedAt
	if ctx.Err() == context.Canceled {
		entry.status = models.QueryStatusCancelled
	} else if err != nil {
		entry.status = models.QueryStatusFailed
		entry.errMessage = err.Error()
		ErrorLog().Printf("user=%s | action=async_query_failed | query_id=%s | error=%s",
			userName, entry.queryID, truncate(err.Error(), 500))
	} else {
		entry.status = models.QueryStatusSucceeded
		entry.result = result
		AuditLog().Printf("user=%s | action=async_query_completed | query_id=%s | rows=%d",
			userName, entry.queryID, result.RowCount)
	}
	entry.mu.Unlock()
}

// SubmitTool schedules a tool execution as an async task.
func (a *AsyncQueryService) SubmitTool(toolName string, params map[string]any, _ int, userName string) (string, error) {
	if a.configService == nil {
		return "", fmt.Errorf("config service not available")
	}
	tool, err := a.configService.GetTool(toolName)
	if err != nil {
		return "", err
	}

	queryID := uuid.NewString()
	now := time.Now()
	ctx, cancel := context.WithCancel(context.Background())

	entry := &asyncQueryEntry{
		queryID:    queryID,
		toolConfig: tool,
		toolParams: params,
		status:     models.QueryStatusPending,
		createdAt:  now,
		userName:   userName,
		cancel:     cancel,
	}

	a.mu.Lock()
	a.queries[queryID] = entry
	a.mu.Unlock()

	AuditLog().Printf("user=%s | action=submit_async_tool | query_id=%s | tool=%s",
		userName, queryID, toolName)

	go a.runTool(ctx, entry)
	return queryID, nil
}

func (a *AsyncQueryService) runTool(ctx context.Context, entry *asyncQueryEntry) {
	entry.mu.Lock()
	entry.status = models.QueryStatusRunning
	startedAt := time.Now()
	entry.startedAt = &startedAt
	tool := entry.toolConfig
	params := entry.toolParams
	userName := entry.userName
	entry.mu.Unlock()

	result, err := a.queryService.ExecuteTemplate(ctx, tool, params, userName)

	entry.mu.Lock()
	completedAt := time.Now()
	entry.completedAt = &completedAt
	if ctx.Err() == context.Canceled {
		entry.status = models.QueryStatusCancelled
	} else if err != nil {
		entry.status = models.QueryStatusFailed
		entry.errMessage = err.Error()
		ErrorLog().Printf("user=%s | action=async_tool_failed | query_id=%s | error=%s",
			userName, entry.queryID, truncate(err.Error(), 500))
	} else {
		entry.status = models.QueryStatusSucceeded
		entry.result = result
		AuditLog().Printf("user=%s | action=async_tool_completed | query_id=%s | rows=%d",
			userName, entry.queryID, result.RowCount)
	}
	entry.mu.Unlock()
}

// GetStatus returns the current status of an async query.
func (a *AsyncQueryService) GetStatus(queryID string) (*models.AsyncQueryStatusResponse, error) {
	a.mu.RLock()
	entry, ok := a.queries[queryID]
	a.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("query not found: %s", queryID)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	resp := &models.AsyncQueryStatusResponse{
		QueryID:     queryID,
		Status:      entry.status,
		CreatedAt:   entry.createdAt,
		StartedAt:   entry.startedAt,
		CompletedAt: entry.completedAt,
	}
	if entry.startedAt != nil && entry.completedAt != nil {
		ms := float64(entry.completedAt.Sub(*entry.startedAt).Microseconds()) / 1000.0
		resp.ExecutionTimeMs = &ms
	}
	if entry.errMessage != "" {
		msg := entry.errMessage
		resp.ErrorMessage = &msg
	}
	if entry.result != nil {
		rc := entry.result.RowCount
		resp.RowCount = &rc
		tr := entry.result.Truncated
		resp.Truncated = &tr
	}
	return resp, nil
}

// GetResult returns the result of a completed async query.
func (a *AsyncQueryService) GetResult(queryID string) (*models.AsyncQueryResult, error) {
	a.mu.RLock()
	entry, ok := a.queries[queryID]
	a.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("query not found: %s", queryID)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.status == models.QueryStatusFailed {
		msg := entry.errMessage
		return &models.AsyncQueryResult{
			QueryID:      queryID,
			Status:       entry.status,
			ErrorMessage: &msg,
		}, nil
	}

	if entry.status != models.QueryStatusSucceeded {
		return nil, fmt.Errorf("query not completed yet: %s (status=%s)", queryID, entry.status)
	}

	rc := entry.result.RowCount
	ms := entry.result.ExecutionTimeMs
	return &models.AsyncQueryResult{
		QueryID:         queryID,
		Status:          entry.status,
		Columns:         entry.result.Columns,
		Rows:            entry.result.Rows,
		RowCount:        &rc,
		ExecutionTimeMs: &ms,
	}, nil
}

// Cancel attempts to cancel a running query.
func (a *AsyncQueryService) Cancel(queryID string) error {
	a.mu.RLock()
	entry, ok := a.queries[queryID]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("query not found: %s", queryID)
	}

	entry.mu.Lock()
	defer entry.mu.Unlock()

	if entry.status != models.QueryStatusRunning && entry.status != models.QueryStatusPending {
		return nil
	}
	if entry.cancel != nil {
		entry.cancel()
	}
	now := time.Now()
	entry.status = models.QueryStatusCancelled
	entry.completedAt = &now
	AuditLog().Printf("user=%s | action=cancel_async_query | query_id=%s", entry.userName, queryID)
	return nil
}
