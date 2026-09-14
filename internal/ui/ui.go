// Package ui provides minimal terminal styling with ANSI escapes. Colors are
// used only on terminals, never with NO_COLOR set or TERM=dumb, and always
// with FORCE_COLOR set.
package ui

import (
	"fmt"
	"io"
	"os"
)

// Printer writes to one stream and styles text for it.
type Printer struct {
	w     io.Writer
	tty   bool
	color bool
}

// New returns a Printer for f, detecting terminal and color support.
func New(f *os.File) *Printer {
	tty := IsTerminal(f)
	return NewWriter(f, tty, colorWanted(tty))
}

// NewWriter returns a Printer with explicit settings.
func NewWriter(w io.Writer, tty, color bool) *Printer {
	return &Printer{w: w, tty: tty, color: color}
}

func colorWanted(tty bool) bool {
	switch {
	case os.Getenv("NO_COLOR") != "", os.Getenv("TERM") == "dumb":
		return false
	case os.Getenv("FORCE_COLOR") != "":
		return true
	}
	return tty
}

// IsTerminal reports whether f is a character device such as a terminal.
func IsTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// TTY reports whether the stream is a terminal.
func (p *Printer) TTY() bool { return p.tty }

func (p *Printer) Printf(format string, a ...any) {
	fmt.Fprintf(p.w, format, a...)
}

func (p *Printer) style(code, s string) string {
	if !p.color || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (p *Printer) Bold(s string) string      { return p.style("1", s) }
func (p *Printer) Dim(s string) string       { return p.style("2", s) }
func (p *Printer) Red(s string) string       { return p.style("31", s) }
func (p *Printer) Green(s string) string     { return p.style("32", s) }
func (p *Printer) Yellow(s string) string    { return p.style("33", s) }
func (p *Printer) Blue(s string) string      { return p.style("34", s) }
func (p *Printer) Magenta(s string) string   { return p.style("35", s) }
func (p *Printer) Cyan(s string) string      { return p.style("36", s) }
func (p *Printer) Highlight(s string) string { return p.style("1;4;33", s) }
