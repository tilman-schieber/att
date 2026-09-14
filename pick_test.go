package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tilman-schieber/att/internal/store"
)

// fakeFzf writes a shell script standing in for fzf.
func fakeFzf(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fzf")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPickFzfParsesKeyAndSelection(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("FAKE_FZF_INPUT", filepath.Join(dir, "input"))
	t.Setenv("FAKE_FZF_ARGS", filepath.Join(dir, "args"))
	fzf := fakeFzf(t, `printf '%s\n' "$@" > "$FAKE_FZF_ARGS"
cat > "$FAKE_FZF_INPUT"
printf 'ctrl-y\n'
sed -n 2p "$FAKE_FZF_INPUT"
`)
	entries := []store.Entry{{Name: "a.pdf"}, {Name: "my notes (1).txt"}}

	e, key, ok, err := pickFzf(fzf, entries, "notes")
	if err != nil || !ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if e.Name != "my notes (1).txt" || key != "ctrl-y" {
		t.Errorf("got %q with key %q", e.Name, key)
	}
	args, _ := os.ReadFile(filepath.Join(dir, "args"))
	for _, want := range []string{"--query=notes", "--expect=ctrl-y", " preview {2}"} {
		if !strings.Contains(string(args), want) {
			t.Errorf("fzf args lack %q:\n%s", want, args)
		}
	}
}

func TestPickFzfCancelled(t *testing.T) {
	fzf := fakeFzf(t, "cat >/dev/null; exit 130\n")
	_, _, ok, err := pickFzf(fzf, []store.Entry{{Name: "a.pdf"}}, "")
	if ok || err != nil {
		t.Errorf("ok=%v err=%v, want cancelled without error", ok, err)
	}
}

func TestStripANSI(t *testing.T) {
	if got := stripANSI("\x1b[2mtoday\x1b[0m\t\x1b[31ma.pdf\x1b[0m"); got != "today\ta.pdf" {
		t.Errorf("got %q", got)
	}
}
