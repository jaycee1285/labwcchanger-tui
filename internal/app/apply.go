package app

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/beevik/etree"
	"github.com/jaycee1285/labwcchanger-tui/internal/theme"
)

type Selections struct {
	OpenboxTheme string
	GtkTheme     string
	IconTheme    string
	KittyTheme   string
	Wallpaper    string
}

func Apply(sel Selections) error {
	if err := updateRcXml(sel); err != nil {
		return err
	}
	if err := updateGSettings(sel); err != nil {
		return err
	}
	if err := updateEnvironment(sel); err != nil {
		return err
	}
	if sel.Wallpaper != "" {
		wpPath := filepath.Join(theme.WallpaperDir(), sel.Wallpaper)
		_ = run("swww", "img", wpPath)
	}
	if sel.KittyTheme != "" {
		if err := applyKittyTheme(sel.KittyTheme); err != nil {
			return err
		}
		if err := updateFuzzelColors(sel.KittyTheme); err != nil {
			return err
		}
	}
	_ = run("labwc", "-r")

	// Waybar doesn't always pick up GTK theme changes unless restarted.
	_ = runNoFail("pkill", "waybar")
	_ = startNoWait("waybar")
	return nil
}

func runNoFail(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	_ = cmd.Run()
	return nil
}

func startNoWait(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return err
	}
	// Don't wait; this is a long-running bar.
	return nil
}

func updateRcXml(sel Selections) error {
	rc := theme.LabwcRcPath()
	if _, err := os.Stat(rc); err != nil {
		return nil // match Flutter: do nothing if missing
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(rc); err != nil {
		return fmt.Errorf("read rc.xml: %w", err)
	}

	if sel.OpenboxTheme != "" {
		for _, el := range doc.FindElements("//theme/name") {
			if el.Parent() != nil && el.Parent().Tag == "theme" {
				el.SetText(sel.OpenboxTheme)
				break
			}
		}
	}
	if sel.IconTheme != "" {
		for _, el := range doc.FindElements("//theme/icon") {
			if el.Parent() != nil && el.Parent().Tag == "theme" {
				el.SetText(sel.IconTheme)
				break
			}
		}
	}
	doc.Indent(2)
	if err := doc.WriteToFile(rc); err != nil {
		return fmt.Errorf("write rc.xml: %w", err)
	}
	return nil
}

func updateGSettings(sel Selections) error {
	if sel.GtkTheme != "" {
		if err := run("gsettings", "set", "org.gnome.desktop.interface", "gtk-theme", sel.GtkTheme); err != nil {
			return err
		}
	}
	if sel.IconTheme != "" {
		if err := run("gsettings", "set", "org.gnome.desktop.interface", "icon-theme", sel.IconTheme); err != nil {
			return err
		}
	}
	return nil
}

func updateEnvironment(sel Selections) error {
	if sel.GtkTheme == "" {
		return nil
	}
	envPath := theme.LabwcEnvPath()
	f, err := os.Open(envPath)
	if err != nil {
		return nil // match Flutter: do nothing if missing
	}
	defer f.Close()

	var out bytes.Buffer
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if len(line) >= 10 && line[:10] == "GTK_THEME=" {
			out.WriteString("GTK_THEME=" + sel.GtkTheme)
			out.WriteByte('\n')
		} else {
			out.WriteString(line)
			out.WriteByte('\n')
		}
	}
	if err := s.Err(); err != nil {
		return fmt.Errorf("read environment: %w", err)
	}
	if err := os.WriteFile(envPath, out.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write environment: %w", err)
	}
	return nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		// include output for debugging
		return fmt.Errorf("%s %v failed: %w\n%s", name, args, err, buf.String())
	}
	return nil
}

var ErrKittyThemeNotFound = errors.New("kitty theme file not found")