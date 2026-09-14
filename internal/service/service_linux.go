//go:build linux

package service

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func unitPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "systemd", "user", UnitName), nil
}

func systemctl(args ...string) (string, error) {
	out, err := exec.Command("systemctl", append([]string{"--user"}, args...)...).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return text, fmt.Errorf("systemctl --user %s: %s", strings.Join(args, " "), text)
	}
	return text, nil
}

// Install writes the user unit, enables it and (re)starts it.
func Install(bin, root string) (Info, error) {
	path, err := unitPath()
	if err != nil {
		return Info{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Info{}, err
	}
	if err := os.WriteFile(path, []byte(systemdUnit(bin, root)), 0o644); err != nil {
		return Info{}, err
	}
	for _, args := range [][]string{{"daemon-reload"}, {"enable", UnitName}, {"restart", UnitName}} {
		if _, err := systemctl(args...); err != nil {
			return Info{}, err
		}
	}

	info, err := Status()
	if err != nil {
		return info, err
	}
	if state, _ := systemctl("is-active", "graphical-session.target"); state != "active" {
		info.Warnings = append(info.Warnings,
			"graphical-session.target is not active, so after a reboot the watcher only starts with a graphical login")
	}
	if env, err := systemctl("show-environment"); err == nil && !hasDisplay(env) {
		info.Warnings = append(info.Warnings,
			"the systemd user manager knows no WAYLAND_DISPLAY or DISPLAY; att will guess from sockets.\n"+
				"For a proper fix let your compositor run: dbus-update-activation-environment --systemd WAYLAND_DISPLAY DISPLAY")
	}
	return info, nil
}

func hasDisplay(env string) bool {
	for _, line := range strings.Split(env, "\n") {
		if strings.HasPrefix(line, "WAYLAND_DISPLAY=") || strings.HasPrefix(line, "DISPLAY=") {
			return true
		}
	}
	return false
}

// Uninstall stops, disables and removes the user unit.
func Uninstall() error {
	path, err := unitPath()
	if err != nil {
		return err
	}
	_, _ = systemctl("disable", "--now", UnitName)
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	_, err = systemctl("daemon-reload")
	return err
}

// Status reports whether the user unit is installed and running.
func Status() (Info, error) {
	path, err := unitPath()
	if err != nil {
		return Info{}, err
	}
	info := Info{UnitPath: path, Logs: "journalctl --user -u att -f"}
	if _, err := os.Stat(path); err == nil {
		info.Installed = true
	}
	out, err := systemctl("show", UnitName, "--property=ActiveState", "--property=MainPID")
	if err != nil {
		return info, nil
	}
	for _, line := range strings.Split(out, "\n") {
		key, value, _ := strings.Cut(line, "=")
		switch key {
		case "ActiveState":
			info.Running = value == "active"
		case "MainPID":
			info.PID, _ = strconv.Atoi(value)
		}
	}
	return info, nil
}
