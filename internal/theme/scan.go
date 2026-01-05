package theme

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)
// dirEntryIsDir returns true for real directories AND symlinks that point to directories.
// NixOS commonly exposes themes/icons under /run/current-system/sw as symlink entries.
func dirEntryIsDir(parent string, e os.DirEntry) bool {
	if e.IsDir() {
		return true
	}
	if e.Type()&os.ModeSymlink == 0 {
		return false
	}
	fi, err := os.Stat(filepath.Join(parent, e.Name())) // follows symlink
	if err != nil {
		return false
	}
	return fi.IsDir()
}


func ScanKittyThemes() []string {
	dir := KittyThemesDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	set := map[string]struct{}{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.ToLower(filepath.Ext(name)) != ".conf" {
			continue
		}
		base := strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
		if base != "" {
			set[base] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ScanOpenboxThemes() []string {
	set := map[string]struct{}{"GTK": {}}
	for _, dir := range ThemeDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !dirEntryIsDir(dir, e) { continue }
			name := e.Name()
			p := filepath.Join(dir, name, "openbox-3", "themerc")
			if _, err := os.Stat(p); err == nil {
				set[name] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ScanGtkThemes() []string {
	set := map[string]struct{}{}
	for _, dir := range ThemeDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !dirEntryIsDir(dir, e) { continue }
			name := e.Name()
			base := filepath.Join(dir, name)
			gtk3css := filepath.Join(base, "gtk-3.0", "gtk.css")
			gtk4css := filepath.Join(base, "gtk-4.0", "gtk.css")
			gtk3dir := filepath.Join(base, "gtk-3.0")
			if exists(gtk3css) || exists(gtk4css) || exists(gtk3dir) {
				set[name] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ScanIconThemes() []string {
	set := map[string]struct{}{}
	for _, dir := range IconDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !dirEntryIsDir(dir, e) { continue }
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			index := filepath.Join(dir, name, "index.theme")
			if exists(index) {
				set[name] = struct{}{}
			}
		}
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ScanWallpapers() []string {
	dir := WallpaperDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}
	out := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".jpg", ".jpeg", ".png", ".webp":
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
