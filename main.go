// Command att manages note attachments in ~/.att.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tilman-schieber/att/internal/store"
	"github.com/tilman-schieber/att/internal/sys"
	"github.com/tilman-schieber/att/internal/ui"
)

var (
	stdout = ui.New(os.Stdout)
	stderr = ui.New(os.Stderr)
)

type helpRow struct{ cmd, args, desc string }

var helpRows = []helpRow{
	{"add", "FILE...", "copy files into the store, print Markdown links"},
	{"watch", "[--notify]", "move files dropped into the inbox into the store"},
	{"drop", "", "open the inbox in the file manager, then watch it"},
	{"list", "", "list attachments, newest first"},
	{"find", "QUERY", "list attachments whose name matches QUERY"},
	{"pick", "[--print] [QUERY]", "choose an attachment interactively, copy its link"},
	{"open", "ID", "open an attachment in its default application"},
	{"path", "ID", "print the full path of an attachment"},
	{"link", "ID", "print the Markdown link of an attachment"},
	{"service", "install", "run the inbox watcher in the background at login"},
	{"service", "uninstall", "stop and remove the background watcher"},
	{"service", "status", "show whether the background watcher runs"},
}

func printUsage(p *ui.Printer) {
	p.Printf("%s %s\n\n", p.Bold("att"), p.Dim("— note attachments as Markdown file:// links"))
	p.Printf("%s\n", p.Bold("Usage"))
	width := 0
	for _, r := range helpRows {
		width = max(width, len(r.cmd)+1+len(r.args))
	}
	for _, r := range helpRows {
		raw, styled := r.cmd, p.Cyan(r.cmd)
		if r.args != "" {
			raw += " " + r.args
			styled += " " + p.Yellow(r.args)
		}
		p.Printf("  %s %s%s  %s\n", p.Dim("att"), styled, strings.Repeat(" ", width-len(raw)), r.desc)
	}
	p.Printf("\n%s\n", p.Bold("Notes"))
	p.Printf("  %s and %s match any part of a stored file name. If no name contains\n", p.Yellow("ID"), p.Yellow("QUERY"))
	p.Printf("  them, their letters are matched in order: %s finds %s.\n", p.Yellow("mtgnts2"), p.Cyan("meeting notes-2.pdf"))
	p.Printf("  %s uses fzf if installed, else a numbered list.\n", p.Cyan("pick"))
	p.Printf("  Files live in %s; set %s to change that, %s to disable colors.\n",
		p.Cyan("~/.att"), p.Cyan("ATT_DIR"), p.Cyan("NO_COLOR"))
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	cmd, args := args[0], args[1:]
	switch cmd {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	}
	s, err := store.Open()
	if err != nil {
		return fail(err)
	}

	switch cmd {
	case "add":
		if len(args) == 0 {
			return usageErr("add needs at least one FILE")
		}
		return add(s, args)
	case "watch":
		flags := flag.NewFlagSet("watch", flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		notify := flags.Bool("notify", false, "")
		if err := flags.Parse(args); err != nil || flags.NArg() != 0 {
			return usageErr("usage: att watch [--notify]")
		}
		return watchInbox(s, *notify)
	case "drop":
		if len(args) != 0 {
			return usageErr("drop takes no arguments")
		}
		if err := s.EnsureDirs(); err != nil {
			return fail(err)
		}
		if err := sys.Open(s.InboxDir()); err != nil {
			warn("could not open file manager: " + err.Error())
		} else {
			status("↗", stderr.Cyan, "opened "+tilde(s.InboxDir()))
		}
		return watchInbox(s, false)
	case "list":
		if len(args) != 0 {
			return usageErr("list takes no arguments")
		}
		entries, err := s.List()
		if err != nil {
			return fail(err)
		}
		if len(entries) == 0 {
			stderr.Printf("%s\n", stderr.Dim("no attachments yet — add one with: att add FILE"))
			return 0
		}
		printEntries(entries, "")
		return 0
	case "find":
		if len(args) == 0 {
			return usageErr("find needs a QUERY")
		}
		query := strings.Join(args, " ")
		entries, err := s.Find(query)
		if err != nil {
			return fail(err)
		}
		if len(entries) == 0 {
			stderr.Printf("%s\n", stderr.Dim(fmt.Sprintf("no attachments match %q", query)))
			return 1
		}
		printEntries(entries, query)
		return 0
	case "pick":
		return pickCmd(s, args)
	case "preview":
		return previewCmd(s, args)
	case "open", "path", "link":
		if len(args) != 1 {
			return usageErr(cmd + " needs exactly one ID")
		}
		return resolveCmd(s, cmd, args[0])
	case "service":
		return serviceCmd(s, args)
	default:
		return usageErr(fmt.Sprintf("unknown command %q", cmd))
	}
}

func add(s *store.Store, files []string) int {
	code := 0
	for _, f := range files {
		e, err := s.Add(f, false)
		if err != nil {
			code = fail(err)
			continue
		}
		if stderr.TTY() {
			status("✓", stderr.Green, "added "+stderr.Bold(e.Name)+renamedNote(f, e))
		}
		stdout.Printf("%s\n", store.Link(e))
	}
	return code
}

func resolveCmd(s *store.Store, cmd, id string) int {
	e, err := s.Resolve(id)
	var amb *store.AmbiguousError
	if errors.As(err, &amb) {
		return printAmbiguous(amb)
	}
	if err != nil {
		return fail(err)
	}
	switch cmd {
	case "open":
		if err := sys.Open(e.Path); err != nil {
			return fail(err)
		}
		if stderr.TTY() {
			status("↗", stderr.Cyan, "opened "+stderr.Bold(e.Name))
		}
	case "path":
		stdout.Printf("%s\n", e.Path)
	default:
		stdout.Printf("%s\n", store.Link(e))
	}
	return 0
}
