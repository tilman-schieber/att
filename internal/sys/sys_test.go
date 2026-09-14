package sys

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func touch(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDisplayVarsRecoversFromSockets(t *testing.T) {
	runtimeDir, x11Dir := t.TempDir(), t.TempDir()
	touch(t, runtimeDir, "bus", "wayland-1.lock", "wayland-1")
	touch(t, x11Dir, "X0")

	unset := func(string) string { return "" }
	got := displayVars(unset, runtimeDir, x11Dir)
	want := []string{"WAYLAND_DISPLAY=wayland-1", "DISPLAY=:0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	set := func(string) string { return "already" }
	if got := displayVars(set, runtimeDir, x11Dir); got != nil {
		t.Errorf("overrode existing variables: %v", got)
	}
}

func TestDisplayVarsNothingFound(t *testing.T) {
	unset := func(string) string { return "" }
	if got := displayVars(unset, t.TempDir(), filepath.Join(t.TempDir(), "missing")); got != nil {
		t.Errorf("got %v, want nothing", got)
	}
}
