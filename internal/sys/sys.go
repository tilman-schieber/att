// Package sys wraps the platform-specific parts: opening files with the
// default application, the clipboard, and desktop notifications.
package sys

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Open opens path (file or directory) with the default application.
func Open(path string) error {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", path).Run()
	}
	// xdg-open can block until the application exits; don't wait for it.
	cmd := exec.Command("xdg-open", path)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// CopyToClipboard puts text on the system clipboard.
func CopyToClipboard(text string) error {
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	env, wayland := guiEnv()
	candidates := [][]string{{"xclip", "-selection", "clipboard"}, {"xsel", "--clipboard", "--input"}, {"wl-copy"}}
	if wayland {
		candidates = append([][]string{{"wl-copy"}}, candidates[:2]...)
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		cmd.Env = env
		return cmd.Run()
	}
	return errors.New("no clipboard tool found (install wl-clipboard, xclip or xsel)")
}

// Notify shows a desktop notification.
func Notify(title, body string) error {
	if runtime.GOOS == "darwin" {
		// Pass texts as arguments so they need no AppleScript escaping.
		return exec.Command("osascript",
			"-e", "on run argv",
			"-e", "display notification (item 2 of argv) with title (item 1 of argv)",
			"-e", "end run",
			title, body).Run()
	}
	cmd := exec.Command("notify-send", "--app-name=att", title, body)
	cmd.Env, _ = guiEnv()
	return cmd.Run()
}

// guiEnv returns the environment for GUI helpers and whether a Wayland
// display is available. Services started by systemd --user often lack
// WAYLAND_DISPLAY and DISPLAY; they are recovered from the sockets the
// compositor or X server created.
func guiEnv() (env []string, wayland bool) {
	extra := displayVars(os.Getenv, os.Getenv("XDG_RUNTIME_DIR"), "/tmp/.X11-unix")
	wayland = os.Getenv("WAYLAND_DISPLAY") != ""
	for _, v := range extra {
		if strings.HasPrefix(v, "WAYLAND_DISPLAY=") {
			wayland = true
		}
	}
	return append(os.Environ(), extra...), wayland
}

func displayVars(getenv func(string) string, runtimeDir, x11Dir string) []string {
	var vars []string
	if getenv("WAYLAND_DISPLAY") == "" && runtimeDir != "" {
		if name := firstNumbered(runtimeDir, "wayland-"); name != "" {
			vars = append(vars, "WAYLAND_DISPLAY="+name)
		}
	}
	if getenv("DISPLAY") == "" {
		if name := firstNumbered(x11Dir, "X"); name != "" {
			vars = append(vars, "DISPLAY=:"+strings.TrimPrefix(name, "X"))
		}
	}
	return vars
}

// firstNumbered returns the first entry in dir named prefix followed by
// digits only (so "wayland-0" matches but "wayland-0.lock" does not).
func firstNumbered(dir, prefix string) string {
	des, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, de := range des {
		rest, ok := strings.CutPrefix(de.Name(), prefix)
		if ok && rest != "" && strings.Trim(rest, "0123456789") == "" {
			return de.Name()
		}
	}
	return ""
}
