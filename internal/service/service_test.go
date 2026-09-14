package service

import (
	"encoding/xml"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestLaunchdPlistIsValidXML(t *testing.T) {
	plist := launchdPlist("/Users/a&b/.local/bin/att", "/Users/a&b/.att", "/Users/a&b/.att/watch.log")
	dec := xml.NewDecoder(strings.NewReader(plist))
	var strs []string
	inString := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("invalid XML: %v\n%s", err, plist)
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			inString = tok.Name.Local == "string"
		case xml.EndElement:
			inString = false
		case xml.CharData:
			if inString {
				strs = append(strs, string(tok))
			}
		}
	}
	for _, want := range []string{Label, "/Users/a&b/.local/bin/att", "watch", "--notify", "/Users/a&b/.att", "/Users/a&b/.att/watch.log"} {
		if !slices.Contains(strs, want) {
			t.Errorf("plist strings %q lack %q", strs, want)
		}
	}
}

func TestSystemdUnitQuoting(t *testing.T) {
	unit := systemdUnit(`/home/a b/100%/$x/att`, "/home/a b/$HOME%")
	for _, want := range []string{
		`ExecStart="/home/a b/100%%/$$x/att" watch --notify`,
		`Environment="ATT_DIR=/home/a b/$HOME%%"`,
		"WantedBy=graphical-session.target",
	} {
		if !strings.Contains(unit, want) {
			t.Errorf("unit lacks %q:\n%s", want, unit)
		}
	}
}
