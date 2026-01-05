package theme

import (
	"os"
	"path/filepath"
)

func HomeDir() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "/home/john"
}

func ThemeDirs() []string {
	h := HomeDir()
	return []string{
		"/usr/share/themes",
		filepath.Join(h, ".local/share/themes"),
		"/run/current-system/sw/share/themes",
		filepath.Join(h, ".nix-profile/share/themes"),
	}
}

func IconDirs() []string {
	h := HomeDir()
	return []string{
		"/usr/share/icons",
		filepath.Join(h, ".local/share/icons"),
		"/run/current-system/sw/share/icons",
		filepath.Join(h, ".nix-profile/share/icons"),
	}
}

func KittyThemesDir() string {
	h := HomeDir()
	return filepath.Join(h, ".config/kitty/themes")
}

func LabwcRcPath() string {
	h := HomeDir()
	return filepath.Join(h, ".config/labwc/rc.xml")
}

func LabwcEnvPath() string {
	h := HomeDir()
	return filepath.Join(h, ".config/labwc/environment")
}

func FuzzelIniPath() string {
	h := HomeDir()
	return filepath.Join(h, ".config/fuzzel/fuzzel.ini")
}

func WallpaperDir() string {
	h := HomeDir()
	return filepath.Join(h, "Pictures/walls")
}
