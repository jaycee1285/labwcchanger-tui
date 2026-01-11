package app

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jaycee1285/labwcchanger-tui/internal/theme"
)

func resolveKittyThemeFile(themeName string) (string, error) {
	dir := theme.KittyThemesDir()
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.ToLower(filepath.Ext(name)) != ".conf" {
				continue
			}
			if strings.TrimSuffix(name, filepath.Ext(name)) == themeName {
				return filepath.Join(dir, name), nil
			}
		}
	}
	fallback := filepath.Join(dir, themeName+".conf")
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}
	return "", ErrKittyThemeNotFound
}

func applyKittyTheme(themeName string) error {
	// `kitten themes` expects the theme NAME, not the file path.
	// Your picker may include ".conf"; Kitty generally wants the base name.
	name := strings.TrimSpace(themeName)
	name = strings.TrimSuffix(name, filepath.Ext(name)) // drops .conf if present

	if name == "" {
		return nil
	}

	return run("kitten", "themes", "--reload-in=all", name)
}

func parseKittyTheme(content string) map[string]string {
	out := map[string]string{}
	s := bufio.NewScanner(strings.NewReader(content))
	// Matches: key  #RRGGBB
	re := regexp.MustCompile(`^([A-Za-z0-9_-]+)\s+#([0-9A-Fa-f]{6})`)
	for s.Scan() {
		line := strings.TrimRight(s.Text(), "\r\n")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		m := re.FindStringSubmatch(trimmed)
		if len(m) == 3 {
			out[m[1]] = strings.ToUpper(m[2])
		}
	}
	return out
}

func parseKittyMeta(content, key string) string {
	// Matches: ## name: Something
	re := regexp.MustCompile(`(?m)^##\s*` + regexp.QuoteMeta(key) + `\s*:\s*(.+)$`)
	m := re.FindStringSubmatch(content)
	if len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func updateFuzzelColors(selectedKittyTheme string) error {
	p, err := resolveKittyThemeFile(selectedKittyTheme)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("read kitty theme: %w", err)
	}
	content := string(b)
	colors := parseKittyTheme(content)

	schemeName := parseKittyMeta(content, "name")
	if schemeName == "" {
		schemeName = selectedKittyTheme
	}
	schemeAuthor := parseKittyMeta(content, "author")
	if schemeAuthor == "" {
		schemeAuthor = "unknown"
	}

	// Same heuristic mapping as Flutter.
	base05 := strings.ToLower(firstNonEmpty(colors["foreground"], colors["cursor"], "FFFFFF"))
	base00 := strings.ToLower(firstNonEmpty(colors["background"], "000000"))
	base01 := strings.ToLower(firstNonEmpty(colors["inactive_tab_background"], colors["selection_background"], strings.ToUpper(base00)))
	base03 := strings.ToLower(firstNonEmpty(colors["inactive_tab_foreground"], colors["color8"], strings.ToUpper(base05)))
	base06 := strings.ToLower(firstNonEmpty(colors["selection_foreground"], colors["foreground"], strings.ToUpper(base05)))
	base0D := strings.ToLower(firstNonEmpty(colors["color4"], colors["active_border_color"], colors["color12"], strings.ToUpper(base05)))

	fuzzelPath := theme.FuzzelIniPath()

	if err := os.MkdirAll(filepath.Dir(fuzzelPath), 0o755); err != nil {
		return fmt.Errorf("mkdir fuzzel dir: %w", err)
	}

	out := strings.Join([]string{
		"## " + schemeName + " theme",
		"## by " + schemeAuthor,
		"",
		"[colors]",
		"background=" + base01 + "f2",
		"text=" + base05 + "ff",
		"match=" + base0D + "ff",
		"selection=" + base03 + "ff",
		"selection-text=" + base06 + "ff",
		"selection-match=" + base0D + "ff",
		"border=" + base0D + "ff",
		"",
	}, "\n")

	if err := os.WriteFile(fuzzelPath, []byte(out), 0o644); err != nil {
		return fmt.Errorf("write fuzzel.ini: %w", err)
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
