# labwcchanger-tui

A Bubble Tea TUI replacement for your Flutter-based LabWC theme changer.

## What it does

Pick and apply:

- GTK theme
- Icon theme
- LabWC/Openbox theme (edits `~/.config/labwc/rc.xml`)
- Kitty theme (`kitten @ set-colors --all --configured`)
- Wallpaper (`swww img`)

It also regenerates `~/.config/fuzzel/fuzzel.ini` from the selected Kitty theme using your BaseXX heuristic mapping.

## Keybindings

- `Tab` / `Shift+Tab`: switch categories
- `↑` / `↓`: navigate
- `/`: filter
- `Enter`: select
- `a`: apply
- `q`: quit

## Notes

- Matches Flutter behavior for missing files: if `~/.config/labwc/rc.xml` or `~/.config/labwc/environment` don’t exist, it won’t create them.
- You asked to skip the `labwc-gtktheme.py` step; this TUI does not call it.

## Build

### With Nix

```bash
nix build -L
./result/bin/labwcchanger-tui
```

On first build you’ll be prompted to replace `vendorHash` in `flake.nix`.

### Without Nix

```bash
go mod download
go build ./...
./labwcchanger-tui
```
