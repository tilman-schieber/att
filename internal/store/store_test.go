package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s := &Store{Root: t.TempDir()}
	if err := s.EnsureDirs(); err != nil {
		t.Fatal(err)
	}
	return s
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCandidate(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"a.pdf", 1, "a.pdf"},
		{"a.pdf", 2, "a-2.pdf"},
		{"x.tar.gz", 3, "x-3.tar.gz"},
		{"README", 2, "README-2"},
		{".bashrc", 2, ".bashrc-2"},
		{"my file.v2.png", 2, "my file.v2-2.png"},
	}
	for _, tt := range tests {
		if got := candidate(tt.name, tt.n); got != tt.want {
			t.Errorf("candidate(%q, %d) = %q, want %q", tt.name, tt.n, got, tt.want)
		}
	}
}

func TestAddCopyAddsSuffixOnClash(t *testing.T) {
	s := newStore(t)
	src := filepath.Join(t.TempDir(), "report.pdf")
	wantNames := []string{"report.pdf", "report-2.pdf", "report-3.pdf"}
	for i, want := range wantNames {
		writeFile(t, src, want)
		e, err := s.Add(src, false)
		if err != nil {
			t.Fatal(err)
		}
		if e.Name != want {
			t.Errorf("add #%d: name %q, want %q", i+1, e.Name, want)
		}
	}
	for _, name := range wantNames {
		if got := readFile(t, filepath.Join(s.StoreDir(), name)); got != name {
			t.Errorf("%s has content %q; earlier file was overwritten", name, got)
		}
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("copy removed source: %v", err)
	}
}

func TestAddMove(t *testing.T) {
	s := newStore(t)
	src := filepath.Join(s.InboxDir(), "shot.png")
	writeFile(t, src, "img")
	e, err := s.Add(src, true)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("source still exists after move: %v", err)
	}
	if got := readFile(t, e.Path); got != "img" {
		t.Errorf("content %q", got)
	}
}

func TestAddRejectsDirectory(t *testing.T) {
	s := newStore(t)
	if _, err := s.Add(t.TempDir(), false); err == nil {
		t.Fatal("expected error for directory")
	}
}

func TestResolve(t *testing.T) {
	s := newStore(t)
	dir := t.TempDir()
	for _, name := range []string{"report.pdf", "report.pdf", "Data.csv"} {
		src := filepath.Join(dir, name)
		writeFile(t, src, "x")
		if _, err := s.Add(src, false); err != nil {
			t.Fatal(err)
		}
	}
	for id, want := range map[string]string{
		"report.pdf": "report.pdf",
		"data.csv":   "Data.csv",
		"-2":         "report-2.pdf",
	} {
		e, err := s.Resolve(id)
		if err != nil {
			t.Errorf("Resolve(%q): %v", id, err)
		} else if e.Name != want {
			t.Errorf("Resolve(%q) = %q, want %q", id, e.Name, want)
		}
	}
	var amb *AmbiguousError
	if _, err := s.Resolve("report"); !errors.As(err, &amb) || len(amb.Matches) != 2 {
		t.Errorf("Resolve(report): want ambiguous with 2 matches, got %v", err)
	}
	if _, err := s.Resolve("nope"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Resolve(nope): want ErrNotFound, got %v", err)
	}
}

func TestListNewestFirst(t *testing.T) {
	s := newStore(t)
	dir := t.TempDir()
	var added []Entry
	for _, name := range []string{"old.txt", "new.txt"} {
		src := filepath.Join(dir, name)
		writeFile(t, src, "x")
		e, err := s.Add(src, false)
		if err != nil {
			t.Fatal(err)
		}
		added = append(added, e)
	}
	past := time.Now().Add(-time.Hour)
	if err := os.Chtimes(added[0].Path, past, past); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(s.StoreDir(), ".DS_Store"), "")

	entries, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[0].Name != "new.txt" || entries[1].Name != "old.txt" {
		t.Errorf("List = %+v", entries)
	}
}

func TestLink(t *testing.T) {
	tests := []struct {
		e    Entry
		want string
	}{
		{
			Entry{Name: "my report (final).pdf", Path: "/home/u/.att/store/my report (final).pdf"},
			"[my report (final).pdf](file:///home/u/.att/store/my%20report%20%28final%29.pdf)",
		},
		{
			Entry{Name: "Grüße [1].PNG", Path: "/x/Grüße [1].PNG"},
			`![Grüße \[1\].PNG](file:///x/Gr%C3%BC%C3%9Fe%20%5B1%5D.PNG)`,
		},
	}
	for _, tt := range tests {
		if got := Link(tt.e); got != tt.want {
			t.Errorf("Link(%q)\n got %s\nwant %s", tt.e.Name, got, tt.want)
		}
	}
}
