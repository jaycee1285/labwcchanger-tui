package theme

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/beevik/etree"
)

type CurrentSettings struct {
	GtkTheme     string
	IconTheme    string
	OpenboxTheme string
}

func LoadCurrentSettings() CurrentSettings {
	cs := CurrentSettings{}
	cs.GtkTheme = strings.Trim(getGsetting("org.gnome.desktop.interface", "gtk-theme"), "'\n ")
	cs.IconTheme = strings.Trim(getGsetting("org.gnome.desktop.interface", "icon-theme"), "'\n ")
	cs.OpenboxTheme = readLabwcTheme()
	return cs
}

func getGsetting(schema, key string) string {
	cmd := exec.Command("gsettings", "get", schema, key)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	_ = cmd.Run()
	return buf.String()
}

func readLabwcTheme() string {
	rc := LabwcRcPath()
	if _, err := os.Stat(rc); err != nil {
		return ""
	}
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(rc); err != nil {
		return ""
	}
	// Match Flutter: first <name> whose parent is <theme>
	for _, el := range doc.FindElements("//theme/name") {
		if el.Parent() != nil && el.Parent().Tag == "theme" {
			return strings.TrimSpace(el.Text())
		}
	}
	return ""
}
