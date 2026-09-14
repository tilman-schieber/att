package sys

import (
	"reflect"
	"testing"
)

func TestParseMdls(t *testing.T) {
	out := `kMDItemDurationSeconds = 83.4
kMDItemKind            = "PDF document"
kMDItemNumberOfPages   = 3
`
	kind, details := parseMdls(out)
	if kind != "PDF document" {
		t.Errorf("kind = %q", kind)
	}
	if want := []string{"3 pages", "1m23s"}; !reflect.DeepEqual(details, want) {
		t.Errorf("details = %q, want %q", details, want)
	}

	kind, details = parseMdls("kMDItemKind = (null)\nkMDItemNumberOfPages = (null)\n")
	if kind != "" || details != nil {
		t.Errorf("null values: kind %q details %q", kind, details)
	}
}

func TestParseFile(t *testing.T) {
	tests := []struct{ out, kind, pages string }{
		{"PDF document, version 1.3, 1 pages\n", "PDF document", "1 page"},
		{"PDF document, version 1.7, 12 pages", "PDF document", "12 pages"},
		{"PNG image data, 1024 x 1024, 8-bit/color RGBA, non-interlaced", "PNG image data", ""},
		{"ASCII text", "ASCII text", ""},
	}
	for _, tt := range tests {
		kind, pages := parseFile(tt.out)
		if kind != tt.kind || pages != tt.pages {
			t.Errorf("parseFile(%q) = %q, %q; want %q, %q", tt.out, kind, pages, tt.kind, tt.pages)
		}
	}
}
