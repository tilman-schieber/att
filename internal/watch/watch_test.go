package watch

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func mustScan(t *testing.T, w *Watcher) {
	t.Helper()
	if err := w.scan(); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanWaitsUntilFileStopsChanging(t *testing.T) {
	dir := t.TempDir()
	var got []string
	w := &Watcher{Dir: dir, Handle: func(p string) error {
		got = append(got, filepath.Base(p))
		return os.Remove(p)
	}}
	path := filepath.Join(dir, "a.txt")
	writeFile(t, path, "x")

	mustScan(t, w)
	if len(got) != 0 {
		t.Fatalf("handled on first sight: %v", got)
	}
	writeFile(t, path, "xyz") // still being written
	mustScan(t, w)
	if len(got) != 0 {
		t.Fatalf("handled while growing: %v", got)
	}
	mustScan(t, w)
	mustScan(t, w)
	if !reflect.DeepEqual(got, []string{"a.txt"}) {
		t.Fatalf("got %v, want exactly one a.txt", got)
	}
}

func TestScanIgnoresHiddenPartialAndDirs(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{".DS_Store", "video.mp4.crdownload", "big.iso.part"} {
		writeFile(t, filepath.Join(dir, name), "x")
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	w := &Watcher{Dir: dir, Handle: func(p string) error {
		t.Errorf("unexpected handle: %s", p)
		return nil
	}}
	mustScan(t, w)
	mustScan(t, w)
}

func TestScanDoesNotRetryFailureUntilFileChanges(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	writeFile(t, path, "x")
	calls := 0
	w := &Watcher{Dir: dir, Handle: func(string) error {
		calls++
		return errors.New("boom")
	}}
	for range 4 {
		mustScan(t, w)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	writeFile(t, path, "changed")
	mustScan(t, w)
	mustScan(t, w)
	if calls != 2 {
		t.Fatalf("calls after change = %d, want 2", calls)
	}
}

func TestAfterBatchOncePerPoll(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a"), "1")
	writeFile(t, filepath.Join(dir, "b"), "2")
	handled, batches := 0, 0
	w := &Watcher{
		Dir:        dir,
		Handle:     func(p string) error { handled++; return os.Remove(p) },
		AfterBatch: func() { batches++ },
	}
	mustScan(t, w)
	mustScan(t, w)
	mustScan(t, w)
	if handled != 2 || batches != 1 {
		t.Fatalf("handled=%d batches=%d, want 2 and 1", handled, batches)
	}
}
