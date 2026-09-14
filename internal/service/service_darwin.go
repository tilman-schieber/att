//go:build darwin

package service

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func unitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", Label+".plist"), nil
}

func domain() string { return fmt.Sprintf("gui/%d", os.Getuid()) }

// Install writes the LaunchAgent and (re)loads it.
func Install(bin, root string) (Info, error) {
	path, err := unitPath()
	if err != nil {
		return Info{}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Info{}, err
	}
	plist := launchdPlist(bin, root, filepath.Join(root, "watch.log"))
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return Info{}, err
	}

	_ = exec.Command("launchctl", "bootout", domain()+"/"+Label).Run()
	// bootout finishes asynchronously; bootstrap fails until it has.
	var out []byte
	for range 20 {
		out, err = exec.Command("launchctl", "bootstrap", domain(), path).CombinedOutput()
		if err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		return Info{}, fmt.Errorf("launchctl bootstrap: %s", strings.TrimSpace(string(out)))
	}

	var info Info
	for range 10 {
		if info, err = Status(); err != nil || info.Running {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	return info, err
}

// Uninstall unloads and removes the LaunchAgent.
func Uninstall() error {
	path, err := unitPath()
	if err != nil {
		return err
	}
	_ = exec.Command("launchctl", "bootout", domain()+"/"+Label).Run()
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

var (
	stateRe  = regexp.MustCompile(`(?m)^\s*state = (\S+)`)
	pidRe    = regexp.MustCompile(`(?m)^\s*pid = (\d+)`)
	stdoutRe = regexp.MustCompile(`(?m)^\s*stdout path = (.+)$`)
)

// Status reports whether the LaunchAgent is installed and running.
func Status() (Info, error) {
	path, err := unitPath()
	if err != nil {
		return Info{}, err
	}
	info := Info{UnitPath: path}
	if _, err := os.Stat(path); err == nil {
		info.Installed = true
	}
	out, err := exec.Command("launchctl", "print", domain()+"/"+Label).Output()
	if err != nil {
		return info, nil // not loaded
	}
	if m := stateRe.FindSubmatch(out); m != nil {
		info.Running = string(m[1]) == "running"
	}
	if m := pidRe.FindSubmatch(out); m != nil {
		info.PID, _ = strconv.Atoi(string(m[1]))
	}
	if m := stdoutRe.FindSubmatch(out); m != nil {
		info.Logs = "tail -f " + strings.TrimSpace(string(m[1]))
	}
	return info, nil
}
