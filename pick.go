package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tilman-schieber/att/internal/store"
	"github.com/tilman-schieber/att/internal/sys"
	"github.com/tilman-schieber/att/internal/ui"
)

func pickCmd(s *store.Store, args []string) int {
	flags := flag.NewFlagSet("pick", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	printName := flags.Bool("print", false, "")
	if err := flags.Parse(args); err != nil {
		return usageErr("usage: att pick [--print] [QUERY]")
	}
	query := strings.Join(flags.Args(), " ")

	entries, err := s.List()
	if err != nil {
		return fail(err)
	}
	if len(entries) == 0 {
		stderr.Printf("%s\n", stderr.Dim("no attachments yet — add one with: att add FILE"))
		return 1
	}

	var (
		e   store.Entry
		key string
		ok  bool
	)
	if fzf, lookErr := exec.LookPath("fzf"); lookErr == nil && stderr.TTY() {
		e, key, ok, err = pickFzf(fzf, entries, query)
	} else {
		e, ok, err = pickPrompt(entries, query)
	}
	var amb *store.AmbiguousError
	if errors.As(err, &amb) {
		return printAmbiguous(amb)
	}
	if err != nil {
		return fail(err)
	}
	if !ok {
		return 1
	}

	text, what := store.Link(e), "link"
	switch {
	case key == "ctrl-y":
		text, what = e.Path, "path"
	case *printName:
		text = e.Name
	}
	stdout.Printf("%s\n", text)
	if *printName {
		return 0
	}
	if err := sys.CopyToClipboard(text); err != nil {
		warn("clipboard: " + err.Error())
	} else if stderr.TTY() {
		status("⧉", stderr.Dim, stderr.Dim("copied "+what+" of ")+stderr.Bold(e.Name))
	}
	return 0
}

// pickFzf runs fzf over the entries. Each line is "<meta>\t<name>"; only the
// name is searched. key is the pressed --expect key, empty for Enter.
func pickFzf(fzf string, entries []store.Entry, query string) (e store.Entry, key string, ok bool, err error) {
	self, err := os.Executable()
	if err != nil {
		return e, "", false, err
	}
	att := shQuote(self)

	p := ui.Colored()
	now := time.Now()
	var input strings.Builder
	for _, entry := range entries {
		meta := fmt.Sprintf("%-11s %8s", friendlyTime(entry.ModTime, now), humanSize(entry.Size))
		// The name field stays unstyled: fzf substitutes it into {2}.
		fmt.Fprintf(&input, "%s\t%s\n", p.Dim(meta), entry.Name)
	}

	cmd := exec.Command(fzf,
		"--ansi",
		"--delimiter=\t",
		"--nth=2",
		"--query="+query,
		"--prompt=att › ",
		"--header=enter: copy link · ctrl-y: copy path · ctrl-o: open",
		"--expect=ctrl-y",
		"--preview="+att+" preview {2}",
		"--preview-window=right,50%,wrap",
		"--bind=ctrl-o:execute-silent("+att+" open {2} >/dev/null 2>&1)",
	)
	cmd.Stdin = strings.NewReader(input.String())
	cmd.Stderr = os.Stderr
	// fzf runs --preview and --bind commands with $SHELL; fix it to sh so the
	// quoting above holds even for fish users.
	cmd.Env = append(os.Environ(), "SHELL=/bin/sh")
	out, err := cmd.Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) && (exit.ExitCode() == 1 || exit.ExitCode() == 130) {
		return e, "", false, nil // no match or cancelled
	}
	if err != nil {
		return e, "", false, fmt.Errorf("fzf: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(lines) < 2 {
		return e, "", false, nil
	}
	_, name, _ := strings.Cut(stripANSI(lines[1]), "\t")
	for _, entry := range entries {
		if entry.Name == name {
			return entry, lines[0], true, nil
		}
	}
	return e, "", false, fmt.Errorf("%w: %q", store.ErrNotFound, name)
}

// pickPrompt is the fallback without fzf: a numbered list of matches.
func pickPrompt(entries []store.Entry, query string) (store.Entry, bool, error) {
	matches := store.Match(entries, query)
	switch {
	case len(matches) == 0:
		return store.Entry{}, false, fmt.Errorf("%w: %q", store.ErrNotFound, query)
	case len(matches) == 1:
		return matches[0], true, nil
	case !stderr.TTY() || !ui.IsTerminal(os.Stdin):
		return store.Entry{}, false, &store.AmbiguousError{ID: query, Matches: matches}
	}

	const limit = 20
	shown := matches[:min(len(matches), limit)]
	for i, e := range shown {
		stderr.Printf("%s  %s\n", stderr.Cyan(fmt.Sprintf("%3d", i+1)), styleName(stderr, e.Name, query))
	}
	if len(matches) > limit {
		stderr.Printf("%s\n", stderr.Dim(fmt.Sprintf("     … %d more; narrow it down with a QUERY", len(matches)-limit)))
	}
	stderr.Printf("%s", stderr.Bold(fmt.Sprintf("pick 1-%d › ", len(shown))))

	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.TrimSpace(line) == "" {
		if err == nil {
			return store.Entry{}, false, nil
		}
		stderr.Printf("\n")
		return store.Entry{}, false, nil
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(line))
	if convErr != nil || n < 1 || n > len(shown) {
		return store.Entry{}, false, fmt.Errorf("not a number between 1 and %d", len(shown))
	}
	return shown[n-1], true, nil
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			i = j
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
