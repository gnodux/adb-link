package config

import (
	"log/slog"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher debounces filesystem events on the config directory and triggers
// ConfigService.Reload on change. Designed for hot-reload during development
// and operations.
type Watcher struct {
	cs       *ConfigService
	w        *fsnotify.Watcher
	debounce time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewWatcher creates a new Watcher pointed at cs.ConfigDir(). The watcher is
// not started until Start is called.
func NewWatcher(cs *ConfigService, debounce time.Duration) (*Watcher, error) {
	if debounce <= 0 {
		debounce = 500 * time.Millisecond
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.Add(cs.ConfigDir()); err != nil {
		_ = w.Close()
		return nil, err
	}
	return &Watcher{
		cs:       cs,
		w:        w,
		debounce: debounce,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}, nil
}

// Start kicks off the watcher goroutine. Returns immediately.
func (h *Watcher) Start() {
	go h.run()
}

// Stop terminates the watcher. Blocks until the goroutine has fully exited.
func (h *Watcher) Stop() {
	select {
	case <-h.stop:
		// already closed
	default:
		close(h.stop)
	}
	<-h.done
	_ = h.w.Close()
}

func (h *Watcher) run() {
	defer close(h.done)

	var debounceTimer *time.Timer
	armDebounce := func() {
		if debounceTimer == nil {
			debounceTimer = time.NewTimer(h.debounce)
		} else {
			if !debounceTimer.Stop() {
				select {
				case <-debounceTimer.C:
				default:
				}
			}
			debounceTimer.Reset(h.debounce)
		}
	}
	debounceC := func() <-chan time.Time {
		if debounceTimer == nil {
			return nil
		}
		return debounceTimer.C
	}

	for {
		select {
		case <-h.stop:
			return
		case ev, ok := <-h.w.Events:
			if !ok {
				return
			}
			if !strings.HasSuffix(ev.Name, ".yaml") {
				continue
			}
			// Treat any meaningful event (write/create/remove/rename) as cause
			// for reload. Chmod alone is ignored.
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			slog.Debug("config file changed", "file", ev.Name, "op", ev.Op.String())
			armDebounce()
		case err, ok := <-h.w.Errors:
			if !ok {
				return
			}
			slog.Warn("config watcher error", "error", err)
		case <-debounceC():
			if err := h.cs.Reload(); err != nil {
				slog.Warn("config hot-reload failed", "error", err)
			} else {
				slog.Info("config hot-reloaded", "dir", h.cs.ConfigDir())
			}
			debounceTimer = nil
		}
	}
}
