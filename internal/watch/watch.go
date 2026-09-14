// Package watch polls a directory and hands off files once they have stopped
// changing, so half-written downloads are not grabbed.
package watch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Watcher polls Dir every Interval.
type Watcher struct {
	Dir      string
	Interval time.Duration
	// Handle is called for each file whose size and mtime were unchanged
	// since the previous poll. It should remove the file from Dir. A file
	// whose Handle failed is not retried until it changes.
	Handle func(path string) error
	// AfterBatch, if set, is called after a poll that handled at least one file.
	AfterBatch func()

	seen map[string]state
}

type state struct {
	size   int64
	mod    time.Time
	failed bool
}

// Run polls until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		if err := w.scan(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (w *Watcher) scan() error {
	if w.seen == nil {
		w.seen = map[string]state{}
	}
	des, err := os.ReadDir(w.Dir)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(des))
	handled := 0
	for _, de := range des {
		name := de.Name()
		if ignored(name) || !de.Type().IsRegular() {
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		present[name] = true
		cur := state{size: info.Size(), mod: info.ModTime()}
		prev, ok := w.seen[name]
		if !ok || prev.size != cur.size || !prev.mod.Equal(cur.mod) {
			w.seen[name] = cur
			continue
		}
		if prev.failed {
			continue
		}
		if err := w.Handle(filepath.Join(w.Dir, name)); err != nil {
			cur.failed = true
			w.seen[name] = cur
			continue
		}
		delete(w.seen, name)
		handled++
	}
	for name := range w.seen {
		if !present[name] {
			delete(w.seen, name)
		}
	}
	if handled > 0 && w.AfterBatch != nil {
		w.AfterBatch()
	}
	return nil
}

var partialSuffixes = []string{".crdownload", ".part", ".download", ".tmp"}

func ignored(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	lower := strings.ToLower(name)
	for _, suf := range partialSuffixes {
		if strings.HasSuffix(lower, suf) {
			return true
		}
	}
	return false
}
