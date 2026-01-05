package theme

import (
	"sort"
	"strings"
)

// Matches your Flutter style availability heuristic.
var stylePatterns = map[string][]string{
	"Nordic Polar":     {"nordic", "polar", "nord", "light"},
	"Nordfox Light":    {"nordfox", "light"},
	"Nordfox Dark":     {"nordfox", "dark"},
	"Gruvbox Light":    {"gruvbox", "light"},
	"Gruvbox Dark":     {"gruvbox", "dark"},
	"Orchis Orange":    {"orchis", "orange"},
	"Kanagawa Light":   {"kanagawa", "light"},
	"Kanagawa Dark":    {"kanagawa", "dark", "dragon"},
	"Catppuccin Latte": {"catppuccin", "latte"},
	"Catppuccin Mocha": {"catppuccin", "mocha"},
	"Juno Mirage":      {"juno", "mirage", "ayu"},
	"Graphite Light":   {"graphite", "light", "wandb"},
	"Graphite Dark":    {"graphite", "dark", "bandw"},
}

// Matches the Flutter applyThemeStyle keyword sets.
var styleApplyKeywords = map[string][]string{
	"Nordic Polar": {
		"nordic-polar", "nordic polar", "nord-light", "nord light",
	},
	"Nordfox Light":    {"nordfox-light", "nordfox light"},
	"Nordfox Dark":     {"nordfox-dark", "nordfox dark"},
	"Gruvbox Light":    {"gruvbox-light", "gruvbox light", "gruvbox", "light"},
	"Gruvbox Dark":     {"gruvbox-dark", "gruvbox dark", "gruvbox", "dark"},
	"Orchis Orange":    {"orchis-orange", "orchis orange", "orchis", "orange", "ayu-light", "ayu light"},
	"Kanagawa Light":   {"kanagawa-light", "kanagawa light", "kanagawa", "light"},
	"Kanagawa Dark":    {"kanagawa-dark", "kanagawa dark", "kanagawa", "dark", "dragon"},
	"Catppuccin Latte": {"catppuccin-latte", "catppuccin latte", "catppuccin", "latte"},
	"Catppuccin Mocha": {"catppuccin-mocha", "catppuccin mocha", "catppuccin", "mocha"},
	"Juno Mirage":      {"juno-mirage", "juno mirage", "ayu-mirage", "ayu mirage", "mirage"},
	"Graphite Light":   {"graphite-light", "graphite light", "graphite", "light", "wandb"},
	"Graphite Dark":    {"graphite-dark", "graphite dark", "graphite", "dark", "bandw"},
}

func AvailableStyles(gtkThemes, wallpapers []string) []string {
	styles := map[string]struct{}{}
	for styleName, keywords := range stylePatterns {
		hasGtk := anyMatch(gtkThemes, keywords, keywordsLenThreshold(keywords))
		hasWall := anyAnyContains(wallpapers, keywords)
		if hasGtk || hasWall {
			styles[styleName] = struct{}{}
		}
	}
	out := make([]string, 0, len(styles))
	for k := range styles {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ApplyStyle(style string, openbox, gtk, icons, kitty, walls []string) (selOpenbox, selGtk, selIcon, selKitty, selWall string) {
	keywords := styleApplyKeywords[style]
	if len(keywords) == 0 {
		keywords = []string{strings.ToLower(style)}
	}
	selOpenbox = BestMatch(openbox, keywords)
	selGtk = BestMatch(gtk, keywords)
	selIcon = BestMatch(icons, keywords)
	selKitty = BestMatch(kitty, keywords)
	selWall = BestMatch(walls, keywords)
	return
}

func BestMatch(items []string, keywords []string) string {
	best := ""
	bestScore := 0
	for _, item := range items {
		itemLower := strings.ToLower(item)
		score := 0
		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			parts := splitParts(kwLower)
			switch {
			case itemLower == kwLower:
				score += 1000
			case strings.HasPrefix(itemLower, kwLower):
				score += 500
			case strings.Contains(itemLower, kwLower):
				score += 300
			default:
				all := true
				for _, p := range parts {
					if p != "" && !strings.Contains(itemLower, p) {
						all = false
						break
					}
				}
				if all {
					score += 200 * len(parts)
				} else {
					for _, p := range parts {
						if p != "" && strings.Contains(itemLower, p) {
							score += 50
						}
					}
				}
			}
		}

		if len(itemLower) > 30 {
			score -= (len(itemLower) - 30) * 2
		}

		if score > bestScore {
			bestScore = score
			best = item
		}
	}
	if bestScore > 0 {
		return best
	}
	return ""
}

func splitParts(s string) []string {
	// same separators as Flutter: space, hyphen, underscore
	seps := func(r rune) bool {
		switch r {
		case ' ', '-', '_', '\t':
			return true
		default:
			return false
		}
	}
	return strings.FieldsFunc(s, seps)
}

func keywordsLenThreshold(keywords []string) int {
	if len(keywords) == 1 {
		return 1
	}
	return 2
}

func anyMatch(items []string, keywords []string, minCount int) bool {
	for _, it := range items {
		lower := strings.ToLower(it)
		count := 0
		for _, kw := range keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				count++
			}
		}
		if count >= minCount {
			return true
		}
	}
	return false
}

func anyAnyContains(items []string, keywords []string) bool {
	for _, it := range items {
		lower := strings.ToLower(it)
		for _, kw := range keywords {
			if strings.Contains(lower, strings.ToLower(kw)) {
				return true
			}
		}
	}
	return false
}
