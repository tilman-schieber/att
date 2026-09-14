package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tilman-schieber/att/internal/service"
	"github.com/tilman-schieber/att/internal/store"
)

func serviceCmd(s *store.Store, args []string) int {
	if len(args) != 1 {
		return usageErr("usage: att service install|uninstall|status")
	}
	switch args[0] {
	case "install":
		bin, err := executablePath()
		if err != nil {
			return fail(err)
		}
		if err := s.EnsureDirs(); err != nil {
			return fail(err)
		}
		info, err := service.Install(bin, s.Root)
		if err != nil {
			return fail(err)
		}
		status("✓", stderr.Green, "installed service for "+stderr.Bold(tilde(bin)))
		printServiceInfo(info)
		return 0
	case "uninstall":
		if err := service.Uninstall(); err != nil {
			return fail(err)
		}
		status("✓", stderr.Green, "service removed")
		return 0
	case "status":
		info, err := service.Status()
		if err != nil {
			return fail(err)
		}
		printServiceInfo(info)
		if !info.Running {
			return 1
		}
		return 0
	default:
		return usageErr(fmt.Sprintf("unknown service command %q (install, uninstall, status)", args[0]))
	}
}

func printServiceInfo(info service.Info) {
	switch {
	case info.Running:
		pid := ""
		if info.PID > 0 {
			pid = stdout.Dim(fmt.Sprintf("  pid %d", info.PID))
		}
		stdout.Printf("%s %s%s\n", stdout.Green("●"), stdout.Bold("running"), pid)
	case info.Installed:
		stdout.Printf("%s %s\n", stdout.Yellow("●"), stdout.Bold("installed, not running"))
	default:
		stdout.Printf("%s %s  %s\n", stdout.Dim("○"), stdout.Bold("not installed"), stdout.Dim("att service install"))
		return
	}
	stdout.Printf("  %s  %s\n", stdout.Dim("unit"), tilde(info.UnitPath))
	if info.Logs != "" {
		stdout.Printf("  %s  %s\n", stdout.Dim("logs"), tilde(info.Logs))
	}
	for _, w := range info.Warnings {
		warn(w)
	}
}

// executablePath returns the resolved path of the running binary, refusing
// temporary `go run` builds that would vanish.
func executablePath() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", err
	}
	if bin, err = filepath.EvalSymlinks(bin); err != nil {
		return "", err
	}
	if strings.Contains(bin, "go-build") {
		return "", errors.New("att runs from a temporary go build; install it first (just install)")
	}
	return bin, nil
}
