package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jaycee1285/labwcchanger-tui/internal/app"
	"github.com/jaycee1285/labwcchanger-tui/internal/theme"
)

type tab int

const (
	tabStyle tab = iota
	tabGtk
	tabIcons
	tabLabwc
	tabKitty
	tabWall
)

var tabNames = []string{"Style", "GTK", "Icons", "LabWC", "Kitty", "Walls"}

type item struct{ title string }

func (i item) Title() string       { return i.title }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return i.title }

type compactDelegate struct {
  normal  lipgloss.Style
  focused lipgloss.Style
}

func newCompactDelegate() compactDelegate {
  return compactDelegate{
    normal:  lipgloss.NewStyle(),
    focused: lipgloss.NewStyle().Reverse(true).Bold(true),
  }
}

func (d compactDelegate) Height() int  { return 1 }
func (d compactDelegate) Spacing() int { return 0 }
func (d compactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
  it, _ := listItem.(item)
  prefix := "• "
  line := prefix + it.title

  if index == m.Index() {
    fmt.Fprint(w, d.focused.Render(line))
    return
  }
  fmt.Fprint(w, d.normal.Render(line))
}


type dataLoadedMsg struct {
	openbox []string
	gtk     []string
	icons   []string
	kitty   []string
	walls   []string
	styles  []string
	current theme.CurrentSettings
}

type applyDoneMsg struct{ err error }

type Model struct {
	active  tab
	lists   map[tab]list.Model
	spinner spinner.Model
	width   int
	height  int

	openbox []string
	gtk     []string
	icons   []string
	kitty   []string
	walls   []string
	styles  []string

	selected app.Selections
	status   string
	applying bool
}

func New() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line
	m := Model{
		active:  tabGtk,
		lists:   map[tab]list.Model{},
		spinner: sp,
		status:  "Loading…",
	}
	del := newCompactDelegate()

for t := tabStyle; t <= tabWall; t++ {
  l := list.New([]list.Item{}, del, 0, 0)
  l.SetShowStatusBar(false)
  l.SetFilteringEnabled(true)
  l.SetShowHelp(false)
  l.SetShowTitle(false) // <- prevents the “second title”
  m.lists[t] = l
}
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadDataCmd())
}

func loadDataCmd() tea.Cmd {
	return func() tea.Msg {
		gtk := theme.ScanGtkThemes()
		walls := theme.ScanWallpapers()
		msg := dataLoadedMsg{
			openbox: theme.ScanOpenboxThemes(),
			gtk:     gtk,
			icons:   theme.ScanIconThemes(),
			kitty:   theme.ScanKittyThemes(),
			walls:   walls,
			styles:  theme.AvailableStyles(gtk, walls),
			current: theme.LoadCurrentSettings(),
		}
		return msg
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m = m.resizeLists()
		return m, nil

	case spinner.TickMsg:
		if m.applying {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case dataLoadedMsg:
		m.openbox, m.gtk, m.icons, m.kitty, m.walls, m.styles = msg.openbox, msg.gtk, msg.icons, msg.kitty, msg.walls, msg.styles

		m.selected.GtkTheme = msg.current.GtkTheme
		m.selected.IconTheme = msg.current.IconTheme
		m.selected.OpenboxTheme = msg.current.OpenboxTheme
		m.status = "Loaded. Tab/Shift+Tab to switch. Enter to select. A to apply. Q to quit."

		m.lists[tabStyle] = rebuildList(m.lists[tabStyle], msg.styles)
		m.lists[tabGtk] = rebuildList(m.lists[tabGtk], msg.gtk)
		m.lists[tabIcons] = rebuildList(m.lists[tabIcons], msg.icons)
		m.lists[tabLabwc] = rebuildList(m.lists[tabLabwc], msg.openbox)
		m.lists[tabKitty] = rebuildList(m.lists[tabKitty], msg.kitty)
		m.lists[tabWall] = rebuildList(m.lists[tabWall], msg.walls)
		// try to position cursors on current selections
		m = m.syncCursorToSelection()
		return m, nil

	case applyDoneMsg:
		m.applying = false
		if msg.err != nil {
			m.status = "Apply failed: " + firstLine(msg.err.Error())
		} else {
			m.status = "Applied."
		}
		return m, nil

	case tea.KeyMsg:
		k := msg.String()
		switch k {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.active = (m.active + 1) % tab(len(tabNames))
			return m, nil
		case "shift+tab":
			m.active = (m.active - 1)
			if m.active < 0 {
				m.active = tab(len(tabNames) - 1)
			}
			return m, nil
		case "a":
			if m.applying {
				return m, nil
			}
			m.applying = true
			m.status = "Applying…"
			return m, tea.Batch(m.spinner.Tick, applyCmd(m.selected))
		case "enter":
			m = m.selectCurrentItem()
			return m, nil
		}
	}

	// forward to active list for navigation/filtering
	l := m.lists[m.active]
	l, cmd = l.Update(msg)
	m.lists[m.active] = l
	return m, cmd
}

func rebuildList(l list.Model, items []string) list.Model {
	lis := make([]list.Item, 0, len(items))
	for _, it := range items {
		lis = append(lis, item{title: it})
	}
	l.SetItems(lis)
	return l
}

func applyCmd(sel app.Selections) tea.Cmd {
	return func() tea.Msg {
		// Skip step 7 (labwc-gtktheme.py) by design.
		err := app.Apply(sel)
		return applyDoneMsg{err: err}
	}
}

func (m Model) selectCurrentItem() Model {
	l := m.lists[m.active]
	it, ok := l.SelectedItem().(item)
	if !ok {
		return m
	}

	switch m.active {
	case tabStyle:
		// Apply style heuristics to fill selections.
		ob, gtk, icon, kitty, wall := theme.ApplyStyle(it.title, m.openbox, m.gtk, m.icons, m.kitty, m.walls)
		if ob != "" {
			m.selected.OpenboxTheme = ob
		}
		if gtk != "" {
			m.selected.GtkTheme = gtk
		}
		if icon != "" {
			m.selected.IconTheme = icon
		}
		if kitty != "" {
			m.selected.KittyTheme = kitty
		}
		if wall != "" {
			m.selected.Wallpaper = wall
		}
		m.status = fmt.Sprintf("Style set: %s", it.title)
		m = m.syncCursorToSelection()
	case tabGtk:
		m.selected.GtkTheme = it.title
		m.status = "GTK: " + it.title
	case tabIcons:
		m.selected.IconTheme = it.title
		m.status = "Icons: " + it.title
	case tabLabwc:
		m.selected.OpenboxTheme = it.title
		m.status = "LabWC: " + it.title
	case tabKitty:
		m.selected.KittyTheme = it.title
		m.status = "Kitty: " + it.title
	case tabWall:
		m.selected.Wallpaper = it.title
		m.status = "Wallpaper: " + it.title
	}
	return m
}

func (m Model) syncCursorToSelection() Model {
	m.lists[tabGtk] = moveCursorTo(m.lists[tabGtk], m.selected.GtkTheme)
	m.lists[tabIcons] = moveCursorTo(m.lists[tabIcons], m.selected.IconTheme)
	m.lists[tabLabwc] = moveCursorTo(m.lists[tabLabwc], m.selected.OpenboxTheme)
	m.lists[tabKitty] = moveCursorTo(m.lists[tabKitty], m.selected.KittyTheme)
	m.lists[tabWall] = moveCursorTo(m.lists[tabWall], m.selected.Wallpaper)
	return m
}

func moveCursorTo(l list.Model, title string) list.Model {
	if title == "" {
		return l
	}
	for idx, it := range l.Items() {
		if ii, ok := it.(item); ok && ii.title == title {
			l.Select(idx)
			break
		}
	}
	return l
}

func (m Model) View() string {
	info := renderSelections(m.selected)
	body := m.renderDashboard()
	status := m.status
	if m.applying {
		status = m.spinner.View() + " " + status
	}

	footer := renderFooter()
	return strings.Join([]string{info, body, footer, "", status}, "\n")
}

var (
  infoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
  panelFocused  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("7")).Padding(0, 0)
  panelInactive = lipgloss.NewStyle().Border(lipgloss.HiddenBorder()).Padding(0, 0)
  buttonStyle   = lipgloss.NewStyle().Bold(true)

  titleActive   = lipgloss.NewStyle().Reverse(true).Bold(true).Padding(0, 0)
  titleInactive = lipgloss.NewStyle().Padding(0, 0)
)


func renderFooter() string {
	return buttonStyle.Render("[A] Apply   [Tab] Next Panel   [Enter] Select   [/] Filter   [Q] Quit")
}

func renderSelections(sel app.Selections) string {
	lines := []string{
		fmt.Sprintf("Selected → GTK: %s | Icons: %s | LabWC: %s | Kitty: %s | Wall: %s",
			emptyDash(sel.GtkTheme),
			emptyDash(sel.IconTheme),
			emptyDash(sel.OpenboxTheme),
			emptyDash(sel.KittyTheme),
			emptyDash(sel.Wallpaper),
		),
		"All panels are visible; Tab changes focus. Apply restarts LabWC and Waybar.",
	}
	return infoStyle.Render(strings.Join(lines, "\n"))
}

func (m Model) resizeLists() Model {
	// Two columns, three panels each. Aim for ~6–7 rows of items per panel.
	w := m.width
	h := m.height
	if w <= 0 || h <= 0 {
		return m
	}

	// Reserve: 2 info lines + footer + status spacer.
	reserved := 5
	available := h - reserved
	if available < 12 {
		available = 12
	}
	panelH := available/3 - 1
	if panelH > 9 {
		panelH = 9
	}
	if panelH < 7 {
		panelH = 7
	}

	colW := (w - 3) / 2
	if colW < 30 {
		colW = w - 2
	}
	listW := colW - 4
	if listW < 20 {
		listW = colW - 2
	}

	for t, l := range m.lists {
		l.SetSize(listW, panelH)
		// compact defaults
		l.SetShowStatusBar(false)
		l.SetShowHelp(false)
		m.lists[t] = l
	}
	return m
}

func (m Model) renderDashboard() string {
	left := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderPanel(tabStyle),
		m.renderPanel(tabGtk),
		m.renderPanel(tabIcons),
	)
	right := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderPanel(tabLabwc),
		m.renderPanel(tabKitty),
		m.renderPanel(tabWall),
	)

	// If terminal is narrow, stack columns vertically.
	if m.width > 0 && m.width < 100 {
		return lipgloss.JoinVertical(lipgloss.Left, left, right)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m Model) renderPanel(t tab) string {
	l := m.lists[t]
	header := tabNames[int(t)]

	// Fill the full panel width so the focused inverted title is easy to spot.
	headerWidth := l.Width() + 2

	if m.active == t {
		view := titleActive.Width(headerWidth).Render(header) + "\n" + l.View()
		return panelFocused.Render(view)
	}
	view := titleInactive.Width(headerWidth).Render(header) + "\n" + l.View()
	return panelInactive.Render(view)
}


func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func firstLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
