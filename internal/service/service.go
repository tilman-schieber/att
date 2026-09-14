// Package service installs `att watch --notify` as a per-user background
// service: a LaunchAgent on macOS, a systemd user unit on Linux.
package service

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

const (
	// Label is the launchd job label.
	Label = "io.github.tilman-schieber.att"
	// UnitName is the systemd user unit.
	UnitName = "att.service"
)

// ErrUnsupported is returned on platforms without launchd or systemd.
var ErrUnsupported = errors.New("services are supported on macOS (launchd) and Linux (systemd) only")

// Info describes the service state.
type Info struct {
	Installed bool
	Running   bool
	PID       int
	UnitPath  string
	Logs      string // shell command to follow the logs
	Warnings  []string
}

func xmlEscape(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func launchdPlist(bin, root, logPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>watch</string>
		<string>--notify</string>
	</array>
	<key>EnvironmentVariables</key>
	<dict>
		<key>ATT_DIR</key>
		<string>%s</string>
	</dict>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%[4]s</string>
	<key>StandardErrorPath</key>
	<string>%[4]s</string>
</dict>
</plist>
`, Label, xmlEscape(bin), xmlEscape(root), xmlEscape(logPath))
}

// The unit is bound to the graphical session so the watcher starts once
// WAYLAND_DISPLAY/DISPLAY exist, which the clipboard and notifications need.
func systemdUnit(bin, root string) string {
	return fmt.Sprintf(`[Unit]
Description=att inbox watcher
PartOf=graphical-session.target
After=graphical-session.target

[Service]
ExecStart=%s watch --notify
Environment=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=graphical-session.target
`, systemdQuote(bin, true), systemdQuote("ATT_DIR="+root, false))
}

// systemdQuote double-quotes a word for a unit file. Specifiers (%) are
// always escaped; $ only on command lines, where variables are expanded.
func systemdQuote(s string, command bool) string {
	pairs := []string{`\`, `\\`, `"`, `\"`, "%", "%%"}
	if command {
		pairs = append(pairs, "$", "$$")
	}
	return `"` + strings.NewReplacer(pairs...).Replace(s) + `"`
}
