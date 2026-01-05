package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jaycee1285/labwcchanger-tui/internal/ui"
)

func main() {
	m := ui.New()
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "labwcchanger-tui error:", err)
		os.Exit(1)
	}
}
