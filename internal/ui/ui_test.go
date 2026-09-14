package ui

import (
	"bytes"
	"testing"
)

func TestStyle(t *testing.T) {
	var buf bytes.Buffer
	on := NewWriter(&buf, true, true)
	off := NewWriter(&buf, true, false)
	if got := on.Bold("x"); got != "\x1b[1mx\x1b[0m" {
		t.Errorf("Bold with color = %q", got)
	}
	if got := off.Bold("x"); got != "x" {
		t.Errorf("Bold without color = %q", got)
	}
	if got := on.Red(""); got != "" {
		t.Errorf("empty string styled: %q", got)
	}
}
