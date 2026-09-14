package main

import (
	"bufio"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/tilman-schieber/att/internal/store"
	"github.com/tilman-schieber/att/internal/sys"
	"github.com/tilman-schieber/att/internal/ui"
)

const previewLines = 300

// previewCmd renders the fzf preview pane for one attachment. Not listed in
// help; `att pick` calls it.
func previewCmd(s *store.Store, args []string) int {
	if len(args) != 1 {
		return usageErr("usage: att preview NAME")
	}
	e, err := s.Resolve(args[0])
	if err != nil {
		return fail(err)
	}
	p := ui.Colored()

	head := readHead(e.Path, 8192)
	mime := http.DetectContentType(head)
	kind, details := sys.Describe(e.Path)
	if kind == "" {
		kind = mime
	}
	if w, h, ok := imageSize(e.Path); ok {
		details = append([]string{fmt.Sprintf("%d × %d px", w, h)}, details...)
	}
	meta := append([]string{kind}, details...)
	meta = append(meta, humanSize(e.Size), friendlyTime(e.ModTime, time.Now()))

	fmt.Println(p.Bold(styleName(p, e.Name, "")))
	fmt.Println(p.Dim(strings.Join(meta, " · ")))
	fmt.Println(p.Dim(tilde(e.Path)))
	if strings.HasPrefix(mime, "text/") && len(head) > 0 {
		fmt.Println()
		printText(e.Path)
	}
	return 0
}

func readHead(path string, n int) []byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	buf := make([]byte, n)
	m, _ := io.ReadFull(f, buf)
	return buf[:m]
}

func imageSize(path string) (width, height int, ok bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0, false
	}
	return cfg.Width, cfg.Height, true
}

// printText prints the start of a text file, highlighted by bat if present.
func printText(path string) {
	if bat, err := exec.LookPath("bat"); err == nil {
		cmd := exec.Command(bat, "--style=plain", "--color=always", "--paging=never",
			fmt.Sprintf("--line-range=:%d", previewLines), path)
		cmd.Stdout = os.Stdout
		if cmd.Run() == nil {
			return
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for i := 0; i < previewLines && sc.Scan(); i++ {
		// Neutralize escape sequences in untrusted file content.
		fmt.Println(strings.ReplaceAll(sc.Text(), "\x1b", "␛"))
	}
}
