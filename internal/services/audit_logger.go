package services

import (
	"fmt"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gnodux/adb-link/internal/config"
)

var (
	auditLogger *FileLogger
	errorLogger *FileLogger
	loggerOnce  sync.Once
)

// FileLogger is a simple file-based logger with daily rotation.
type FileLogger struct {
	mu         sync.Mutex
	name       string
	dir        string
	currentDay string
}

// SetupAuditLogging configures audit and error loggers.
func SetupAuditLogging(logDir string) {
	loggerOnce.Do(func() {
		auditLogger = &FileLogger{name: "audit", dir: logDir}
		errorLogger = &FileLogger{name: "error", dir: logDir}
	})
}

// AuditLog returns the global audit logger.
func AuditLog() *FileLogger { return auditLogger }

// ErrorLog returns the global error logger.
func ErrorLog() *FileLogger { return errorLogger }

// Info writes an audit log line.
func (l *FileLogger) Info(msg string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	writeLine(l.dir, l.name, "INFO", msg)
}

// Error writes an error log line.
func (l *FileLogger) Error(msg string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	writeLine(l.dir, l.name, "ERROR", msg)
}

// Printf writes an info log line using printf-style formatting.
func (l *FileLogger) Printf(format string, args ...any) {
	if l == nil {
		return
	}
	l.Info(fmt.Sprintf(format, args...))
}

// Errorf writes an error log line using printf-style formatting.
func (l *FileLogger) Errorf(format string, args ...any) {
	if l == nil {
		return
	}
	l.Error(fmt.Sprintf(format, args...))
}

func writeLine(dir, name, level, msg string) {
	if dir == "" {
		dir = "logs"
	}
	_ = ensureDir(dir)
	filename := filepath.Join(dir, name+".log")
	f, err := openAppend(filename)
	if err != nil {
		return
	}
	defer f.Close()
	ts := time.Now().Format("2006-01-02 15:04:05.000")
	_, _ = f.WriteString(ts + " | " + level + " | " + msg + "\n")
}

// truncateString returns a string truncated to n bytes.
func truncateString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// itoa wraps strconv.Itoa for inline usage.
func itoa(i int) string { return strconv.Itoa(i) }

// SettingsRef is used to avoid import cycles in callers.
type SettingsRef = config.Settings
