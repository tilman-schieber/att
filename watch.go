package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/tilman-schieber/att/internal/store"
	"github.com/tilman-schieber/att/internal/sys"
	"github.com/tilman-schieber/att/internal/watch"
)

func watchInbox(s *store.Store, notify bool) int {
	if err := s.EnsureDirs(); err != nil {
		return fail(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var added []store.Entry
	w := &watch.Watcher{
		Dir:      s.InboxDir(),
		Interval: time.Second,
		Handle: func(path string) error {
			e, err := s.Add(path, true)
			if err != nil {
				fail(fmt.Errorf("could not add %s: %w", filepath.Base(path), err))
				if notify {
					_ = sys.Notify("att", "Could not add "+filepath.Base(path))
				}
				return err
			}
			status("✓", stderr.Green, "added "+stderr.Bold(e.Name)+renamedNote(path, e))
			stdout.Printf("%s\n", store.Link(e))
			added = append(added, e)
			return nil
		},
		AfterBatch: func() {
			links := make([]string, len(added))
			for i, e := range added {
				links[i] = store.Link(e)
			}
			clipErr := sys.CopyToClipboard(strings.Join(links, "\n"))
			if clipErr != nil {
				warn("clipboard: " + clipErr.Error())
			} else {
				status("⧉", stderr.Dim, stderr.Dim(fmt.Sprintf("copied %d %s to clipboard", len(links), plural(len(links), "link"))))
			}
			if notify {
				_ = sys.Notify("att", notification(added, clipErr == nil))
			}
			added = added[:0]
		},
	}

	logTimestamps = !stderr.TTY()
	hint := ""
	if stderr.TTY() {
		hint = stderr.Dim("  Ctrl-C to stop")
	}
	status("●", stderr.Green, "watching "+stderr.Bold(tilde(s.InboxDir()))+hint)
	if err := w.Run(ctx); err != nil {
		return fail(err)
	}
	if stderr.TTY() {
		stderr.Printf("\r%s\n", stderr.Dim("stopped"))
	}
	return 0
}

func notification(added []store.Entry, copied bool) string {
	what := added[0].Name
	if len(added) > 1 {
		what = fmt.Sprintf("%d attachments", len(added))
	}
	switch {
	case !copied:
		return what + " added"
	case len(added) == 1:
		return what + " — link copied"
	default:
		return what + " — links copied"
	}
}
