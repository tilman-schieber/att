package sys

import (
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Describe returns a human-readable kind for a file (e.g. "PDF document")
// and details such as page count or duration. It uses Spotlight metadata on
// macOS and file(1) everywhere. Spotlight skips hidden directories like
// ~/.att, so for those mdls yields the kind only and file(1) fills in pages.
func Describe(path string) (kind string, details []string) {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("mdls",
			"-name", "kMDItemKind",
			"-name", "kMDItemNumberOfPages",
			"-name", "kMDItemDurationSeconds",
			path).Output()
		if err == nil {
			kind, details = parseMdls(string(out))
		}
	}
	if out, err := exec.Command("file", "-b", path).Output(); err == nil {
		fileKind, pages := parseFile(string(out))
		if kind == "" {
			kind = fileKind
		}
		if pages != "" && !hasPages(details) {
			details = append([]string{pages}, details...)
		}
	}
	return kind, details
}

func parseMdls(out string) (kind string, details []string) {
	vals := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(line, "=")
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if ok && value != "(null)" {
			vals[strings.TrimSpace(key)] = value
		}
	}
	if pages := vals["kMDItemNumberOfPages"]; pages != "" {
		details = append(details, pagesLabel(pages))
	}
	if secs, err := strconv.ParseFloat(vals["kMDItemDurationSeconds"], 64); err == nil {
		details = append(details, time.Duration(secs*float64(time.Second)).Round(time.Second).String())
	}
	return vals["kMDItemKind"], details
}

// parseFile reads file(1) output such as "PDF document, version 1.3, 2 pages".
func parseFile(out string) (kind, pages string) {
	parts := strings.Split(strings.TrimSpace(out), ", ")
	for _, part := range parts[1:] {
		fields := strings.Fields(part)
		if len(fields) == 2 && (fields[1] == "pages" || fields[1] == "page") {
			if _, err := strconv.Atoi(fields[0]); err == nil {
				pages = pagesLabel(fields[0])
			}
		}
	}
	return parts[0], pages
}

func pagesLabel(n string) string {
	if n == "1" {
		return "1 page"
	}
	return n + " pages"
}

func hasPages(details []string) bool {
	for _, d := range details {
		if strings.HasSuffix(d, " page") || strings.HasSuffix(d, " pages") {
			return true
		}
	}
	return false
}
