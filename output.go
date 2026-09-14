package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tilman-schieber/att/internal/store"
	"github.com/tilman-schieber/att/internal/ui"
)

// logTimestamps prefixes status lines with the time; set by a watcher whose
// stderr is not a terminal, i.e. the service log.
var logTimestamps bool

// status prints a progress line on stderr.
func status(symbol string, paint func(string) string, msg string) {
	if logTimestamps {
		stderr.Printf("%s %s %s\n", time.Now().Format("2006-01-02 15:04:05"), symbol, msg)
		return
	}
	stderr.Printf("%s %s\n", paint(symbol), msg)
}

func warn(msg string) {
	status("!", stderr.Yellow, stderr.Yellow(msg))
}

func fail(err error) int {
	status("✗", stderr.Red, stderr.Red(stderr.Bold("error:"))+" "+err.Error())
	return 1
}

func usageErr(msg string) int {
	fail(fmt.Errorf("%s", msg))
	stderr.Printf("%s\n", stderr.Dim("run 'att help' for usage"))
	return 2
}

func printAmbiguous(amb *store.AmbiguousError) int {
	fail(amb)
	for _, m := range amb.Matches {
		stderr.Printf("  %s %s\n", stderr.Dim("·"), m.Name)
	}
	stderr.Printf("%s\n", stderr.Dim("use more of the name, or choose with: att pick "+amb.ID))
	return 1
}

// renamedNote explains a suffixed name, e.g. " (report.pdf was taken)".
func renamedNote(src string, e store.Entry) string {
	orig := filepath.Base(src)
	if orig == e.Name {
		return ""
	}
	return stderr.Dim(" (" + orig + " was taken)")
}

func printEntries(entries []store.Entry, query string) {
	if !stdout.TTY() {
		for _, e := range entries {
			stdout.Printf("%s  %8s  %s\n", e.ModTime.Format("2006-01-02 15:04"), humanSize(e.Size), e.Name)
		}
		return
	}
	now := time.Now()
	var total int64
	for _, e := range entries {
		total += e.Size
		stdout.Printf("%s  %s  %s\n",
			stdout.Dim(fmt.Sprintf("%-11s", friendlyTime(e.ModTime, now))),
			fmt.Sprintf("%8s", humanSize(e.Size)),
			styleName(stdout, e.Name, query))
	}
	stdout.Printf("%s\n", stdout.Dim(fmt.Sprintf("%d %s · %s", len(entries), plural(len(entries), "attachment"), humanSize(total))))
}

// styleName colors a name by file type, or highlights a substring match of
// query. Fuzzy matches are shown unstyled.
func styleName(p *ui.Printer, name, query string) string {
	if query != "" {
		lowerName, lowerQuery := strings.ToLower(name), strings.ToLower(query)
		i := strings.Index(lowerName, lowerQuery)
		if i < 0 || len(lowerName) != len(name) {
			return name
		}
		j := i + len(lowerQuery)
		return name[:i] + p.Highlight(name[i:j]) + name[j:]
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".heic":
		return p.Magenta(name)
	case ".pdf":
		return p.Red(name)
	case ".zip", ".gz", ".tgz", ".xz", ".bz2", ".zst", ".7z", ".rar":
		return p.Yellow(name)
	case ".mp4", ".mov", ".mkv", ".webm", ".mp3", ".m4a", ".wav", ".flac":
		return p.Blue(name)
	case ".md", ".txt", ".csv", ".json", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".odt":
		return p.Cyan(name)
	}
	return name
}

func friendlyTime(t, now time.Time) string {
	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()
	switch {
	case ty == ny && tm == nm && td == nd:
		return "today " + t.Format("15:04")
	case t.Before(now) && now.Sub(t) < 6*24*time.Hour:
		return t.Format("Mon 15:04")
	case ty == ny:
		return t.Format("Jan 02")
	default:
		return t.Format("2006-01-02")
	}
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// tilde shortens paths under the home directory to ~/...
func tilde(s string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return s
	}
	s = strings.ReplaceAll(s, home+string(filepath.Separator), "~"+string(filepath.Separator))
	if s == home {
		return "~"
	}
	return s
}
