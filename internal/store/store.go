// Package store manages the attachment directory: an inbox drop folder and a
// flat store of files that keep their original names. The filesystem is the
// only source of truth.
package store

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// ErrNotFound is returned by Resolve when no attachment matches.
var ErrNotFound = errors.New("no matching attachment")

// AmbiguousError is returned by Resolve when an ID matches several attachments.
type AmbiguousError struct {
	ID      string
	Matches []Entry
}

func (e *AmbiguousError) Error() string {
	return fmt.Sprintf("%q matches %d attachments", e.ID, len(e.Matches))
}

// Store is an attachment directory rooted at Root.
type Store struct {
	Root string
}

// Entry is one stored attachment.
type Entry struct {
	Name    string
	Path    string
	Size    int64
	ModTime time.Time
}

// Open returns the store at $ATT_DIR, or ~/.att if unset.
func Open() (*Store, error) {
	if dir := os.Getenv("ATT_DIR"); dir != "" {
		abs, err := filepath.Abs(dir)
		if err != nil {
			return nil, err
		}
		return &Store{Root: abs}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &Store{Root: filepath.Join(home, ".att")}, nil
}

func (s *Store) InboxDir() string { return filepath.Join(s.Root, "inbox") }
func (s *Store) StoreDir() string { return filepath.Join(s.Root, "store") }

// EnsureDirs creates the inbox and store directories if needed.
func (s *Store) EnsureDirs() error {
	for _, dir := range []string{s.InboxDir(), s.StoreDir()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// Add copies (or moves) src into the store under its own name. If that name
// is taken, a numeric suffix is added: report.pdf, report-2.pdf, ...
// Existing files are never overwritten.
func (s *Store) Add(src string, move bool) (Entry, error) {
	info, err := os.Stat(src)
	if err != nil {
		return Entry{}, err
	}
	if !info.Mode().IsRegular() {
		return Entry{}, fmt.Errorf("%s: not a regular file", src)
	}
	if err := s.EnsureDirs(); err != nil {
		return Entry{}, err
	}
	f, dst, err := reserve(s.StoreDir(), filepath.Base(src))
	if err != nil {
		return Entry{}, err
	}

	if move {
		f.Close()
		err = os.Rename(src, dst)
		if errors.Is(err, syscall.EXDEV) {
			err = copyAcrossDevices(dst, src)
		}
	} else {
		err = copyInto(f, src)
	}
	if err != nil {
		os.Remove(dst)
		return Entry{}, err
	}

	// Stamp with the ingestion time so `list` shows newest additions first.
	now := time.Now()
	_ = os.Chtimes(dst, now, now)
	info, err = os.Stat(dst)
	if err != nil {
		return Entry{}, err
	}
	return Entry{Name: filepath.Base(dst), Path: dst, Size: info.Size(), ModTime: info.ModTime()}, nil
}

// reserve atomically creates the first free candidate name in dir.
func reserve(dir, name string) (*os.File, string, error) {
	for n := 1; ; n++ {
		path := filepath.Join(dir, candidate(name, n))
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, path, err
	}
}

func candidate(name string, n int) string {
	if n == 1 {
		return name
	}
	stem, ext := splitExt(name)
	return fmt.Sprintf("%s-%d%s", stem, n, ext)
}

var doubleExts = []string{".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst"}

func splitExt(name string) (stem, ext string) {
	lower := strings.ToLower(name)
	for _, de := range doubleExts {
		if strings.HasSuffix(lower, de) && len(name) > len(de) {
			i := len(name) - len(de)
			return name[:i], name[i:]
		}
	}
	ext = filepath.Ext(name)
	if ext == name { // dotfile such as ".bashrc"
		return name, ""
	}
	return name[:len(name)-len(ext)], ext
}

// copyInto copies src into dst and closes dst.
func copyInto(dst *os.File, src string) error {
	in, err := os.Open(src)
	if err != nil {
		dst.Close()
		return err
	}
	defer in.Close()
	if _, err := io.Copy(dst, in); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

func copyAcrossDevices(dst, src string) error {
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if err := copyInto(f, src); err != nil {
		return err
	}
	return os.Remove(src)
}

// List returns all stored attachments, newest first.
func (s *Store) List() ([]Entry, error) {
	des, err := os.ReadDir(s.StoreDir())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var entries []Entry
	for _, de := range des {
		if strings.HasPrefix(de.Name(), ".") || !de.Type().IsRegular() {
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		entries = append(entries, Entry{
			Name:    de.Name(),
			Path:    filepath.Join(s.StoreDir(), de.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if !entries[i].ModTime.Equal(entries[j].ModTime) {
			return entries[i].ModTime.After(entries[j].ModTime)
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

// Find returns attachments whose name contains query (case-insensitive).
func (s *Store) Find(query string) ([]Entry, error) {
	entries, err := s.List()
	if err != nil {
		return nil, err
	}
	return filterName(entries, query), nil
}

func filterName(entries []Entry, query string) []Entry {
	q := strings.ToLower(query)
	var out []Entry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name), q) {
			out = append(out, e)
		}
	}
	return out
}

// Resolve finds one attachment by exact name, case-insensitive name, or
// unique case-insensitive substring, in that order.
func (s *Store) Resolve(id string) (Entry, error) {
	entries, err := s.List()
	if err != nil {
		return Entry{}, err
	}
	for _, e := range entries {
		if e.Name == id {
			return e, nil
		}
	}
	var folded []Entry
	for _, e := range entries {
		if strings.EqualFold(e.Name, id) {
			folded = append(folded, e)
		}
	}
	if len(folded) == 1 {
		return folded[0], nil
	}
	matches := filterName(entries, id)
	switch len(matches) {
	case 0:
		return Entry{}, fmt.Errorf("%w: %q", ErrNotFound, id)
	case 1:
		return matches[0], nil
	default:
		return Entry{}, &AmbiguousError{ID: id, Matches: matches}
	}
}

var labelEscaper = strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`)

var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true,
}

// Link renders a Markdown file:// link; images become embeds.
func Link(e Entry) string {
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(e.Path)}
	prefix := ""
	if imageExts[strings.ToLower(filepath.Ext(e.Name))] {
		prefix = "!"
	}
	return fmt.Sprintf("%s[%s](%s)", prefix, labelEscaper.Replace(e.Name), u.String())
}
