# att

Small CLI for note attachments on macOS and Linux. Files live in one central
directory and notes link to them with plain Markdown `file://` links.

```
~/.att/
├── inbox/      drop folder, emptied by `att watch`
├── store/      managed attachments, flat, original file names
└── watch.log   service log (macOS)
```

No database, no index: the filesystem is the source of truth. Files keep their
names; if a name is taken, a suffix is added (`report.pdf`, `report-2.pdf`, …).
Nothing is ever overwritten.

## Install

```sh
just install    # = GOBIN=~/.local/bin go install .
```

`~/.local/bin` must be on `PATH`. Requires Go 1.22+, no dependencies. On Linux,
clipboard support needs `wl-copy`, `xclip` or `xsel`; notifications need
`notify-send`; opening files uses `xdg-open`.

## Usage

```sh
att add "meeting notes.pdf"   # [meeting notes.pdf](file:///Users/me/.att/store/meeting%20notes.pdf)
att add shot.png              # ![shot.png](file:///Users/me/.att/store/shot.png)
att watch                     # move files dropped into inbox/, print links, copy them to clipboard
att watch --notify            # same, plus a desktop notification per batch
att drop                      # open inbox/ in Finder / file manager, then watch
att list                      # newest first: date, size, name
att find meeting              # case-insensitive substring search
att open report-2             # open in default application
att path report-2             # print absolute path
att link report-2             # print the Markdown link again
```

`ID` is a stored file name or any unique part of it; ambiguous IDs list the
candidates. Images (png, jpg, gif, webp, svg) get `![…](…)` embed links.

`att watch` waits until a file's size and mtime stop changing before ingesting
it, and ignores dotfiles, directories and partial downloads (`.crdownload`,
`.part`, `.download`, `.tmp`). Several files dropped at once are copied to the
clipboard together, one link per line.

Output is colored on terminals only. Links always go to stdout unstyled;
status lines go to stderr, so `att add x | pbcopy` stays clean. `NO_COLOR`
disables colors, `FORCE_COLOR` enables them. `ATT_DIR` changes the root
directory.

## Background service

```sh
att service install     # start `att watch --notify` now and at every login
att service status
att service uninstall
```

Install again after moving or rebuilding the binary. The service records the
current `ATT_DIR` and the absolute binary path.

**macOS** — a LaunchAgent at
`~/Library/LaunchAgents/io.github.tilman-schieber.att.plist`, logging to
`~/.att/watch.log`. Notifications appear as coming from *Script Editor*; allow
them in System Settings → Notifications on first use.

**Linux** — a systemd user unit at `~/.config/systemd/user/att.service`,
logs via `journalctl --user -u att -f`. It is bound to
`graphical-session.target`, so it starts once the desktop is up and stops at
logout. The clipboard tools need `WAYLAND_DISPLAY` or `DISPLAY`:

- GNOME and KDE Plasma export them to systemd automatically.
- Sway, Hyprland and other compositors need, in their config:
  `exec dbus-update-activation-environment --systemd WAYLAND_DISPLAY DISPLAY XDG_CURRENT_DESKTOP`
  (not needed when the session is started with `uwsm`).
- As a fallback att looks for a `wayland-N` socket in `$XDG_RUNTIME_DIR` and
  an `XN` socket in `/tmp/.X11-unix`.

`att service install` warns when the session target or display variables are
missing.
