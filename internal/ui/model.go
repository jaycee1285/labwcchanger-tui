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

// Layout constraints
const (
	maxWidth  = 70
	maxHeight = 30
)

type tab int

const (
	tabStyle tab = iota
	tabGtk
	tabIcons
	tabLabwc
	tabKitty
	tabWall
	tabCount
)

var tabNames = []string{"Style", "GTK", "Icons", "LabWC", "Kitty", "Walls"}

type item struct{ title string }

func (i item) Title() string       { return i.title }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return i.title }

// Compact delegate for items inside expanded panels
type compactDelegate struct {
	normal  lipgloss.Style
	focused lipgloss.Style
}

func newCompactDelegate() compactDelegate {
	return compactDelegate{
		normal:  lipgloss.NewStyle(),
		focused: lipgloss.NewStyle().Bold(true).Reverse(true),
	}
}

func (d compactDelegate) Height() int                                 { return 1 }
func (d compactDelegate) Spacing() int                                { return 0 }
func (d compactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd   { return nil }

func (d compactDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	it, _ := listItem.(item)
	if index == m.Index() {
		line := "  ▸ " + it.title
		fmt.Fprint(w, d.focused.Render(line))
		return
	}
	line := "    " + it.title
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
	active    tab       // Currently focused panel (title row)
	expanded  tab       // Which panel is expanded (-1 = none)
	inList    bool      // True when navigating inside an expanded list
	lists     map[tab]list.Model
	spinner   spinner.Model
	width     int
	height    int

	openbox []string
	gtk     []string
	icons   []string
	kitty   []string
	walls   []string
	styles  []string

	selected app.Selections
	status   string
	applying bool
	loaded   bool
}

func New() Model {
	sp := spinner.New()
	sp.Spinner = spinner.Line
	m := Model{
		active:   tabStyle,
		expanded: -1, // Nothing expanded initially
		inList:   false,
		lists:    map[tab]list.Model{},
		spinner:  sp,
		status:   "Loading…",
	}
	del := newCompactDelegate()

	for t := tabStyle; t < tabCount; t++ {
		l := list.New([]list.Item{}, del, maxWidth-6, 8)
		l.SetShowStatusBar(false)
		l.SetFilteringEnabled(true)
		l.SetShowHelp(false)
		l.SetShowTitle(false)
		l.SetShowPagination(true)
		l.KeyMap.Quit.SetEnabled(false) // We handle quit ourselves
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
		m.width = min(msg.Width, maxWidth)
		m.height = min(msg.Height, maxHeight)
		m = m.resizeLists()
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case dataLoadedMsg:
		m.openbox, m.gtk, m.icons, m.kitty, m.walls, m.styles = msg.openbox, msg.gtk, msg.icons, msg.kitty, msg.walls, msg.styles

		m.selected.GtkTheme = msg.current.GtkTheme
		m.selected.IconTheme = msg.current.IconTheme
		m.selected.OpenboxTheme = msg.current.OpenboxTheme
		m.status = "Ready"
		m.loaded = true

		m.lists[tabStyle] = rebuildList(m.lists[tabStyle], msg.styles)
		m.lists[tabGtk] = rebuildList(m.lists[tabGtk], msg.gtk)
		m.lists[tabIcons] = rebuildList(m.lists[tabIcons], msg.icons)
		m.lists[tabLabwc] = rebuildList(m.lists[tabLabwc], msg.openbox)
		m.lists[tabKitty] = rebuildList(m.lists[tabKitty], msg.kitty)
		m.lists[tabWall] = rebuildList(m.lists[tabWall], msg.walls)
		m = m.syncCursorToSelection()
		return m, nil

	case applyDoneMsg:
		m.applying = false
		if msg.err != nil {
			m.status = "Apply failed: " + firstLine(msg.err.Error())
		} else {
			m.status = "Applied successfully!"
		}
		return m, nil

	case tea.KeyMsg:
		k := msg.String()

		// Global keys
		switch k {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "a":
			if m.applying {
				return m, nil
			}
			m.applying = true
			m.status = "Applying…"
			return m, tea.Batch(m.spinner.Tick, applyCmd(m.selected))
		}

		// Navigation depends on whether we're in a list or at panel titles
		if m.inList && m.expanded >= 0 {
			switch k {
			case "left", "esc":
				// Collapse and return to panel navigation
				m.inList = false
				return m, nil
			case "enter":
				m = m.selectCurrentItem()
				return m, nil
			case "up", "down", "j", "k", "pgup", "pgdown", "home", "end":
				// Forward to list
				l := m.lists[m.expanded]
				l, cmd = l.Update(msg)
				m.lists[m.expanded] = l
				return m, cmd
			default:
				// Forward filtering keys to list
				l := m.lists[m.expanded]
				l, cmd = l.Update(msg)
				m.lists[m.expanded] = l
				return m, cmd
			}
		} else {
			// Navigating panel titles
			switch k {
			case "up", "k":
				if m.active > 0 {
					m.active--
				}
				return m, nil
			case "down", "j":
				if m.active < tabCount-1 {
					m.active++
				}
				return m, nil
			case "right", "enter", "l":
				// Expand the active panel and enter list mode
				m.expanded = m.active
				m.inList = true
				return m, nil
			case "left", "h":
				// Collapse if this panel is expanded
				if m.expanded == m.active {
					m.expanded = -1
				}
				return m, nil
			}
		}
	}

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
		err := app.Apply(sel)
		return applyDoneMsg{err: err}
	}
}

func (m Model) selectCurrentItem() Model {
	if m.expanded < 0 {
		return m
	}
	l := m.lists[m.expanded]
	it, ok := l.SelectedItem().(item)
	if !ok {
		return m
	}

	switch m.expanded {
	case tabStyle:
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
		m.status = fmt.Sprintf("Style applied: %s", it.title)
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

func (m Model) resizeLists() Model {
	if m.width <= 0 || m.height <= 0 {
		return m
	}
	// Calculate available height for expanded list
	// Header (title + border) + selections (6 lines) + panel titles + status
	listHeight := m.height - 16
	if listHeight < 5 {
		listHeight = 5
	}
	if listHeight > 12 {
		listHeight = 12
	}

	listWidth := m.width - 6
	if listWidth < 30 {
		listWidth = 30
	}

	for t, l := range m.lists {
		l.SetSize(listWidth, listHeight)
		m.lists[t] = l
	}
	return m
}

// ─────────────────────────────────────────────────────────────────────────────
// View rendering
// ─────────────────────────────────────────────────────────────────────────────

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	panelTitleFocused = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("7"))

	panelTitleNormal = lipgloss.NewStyle().
				Foreground(lipgloss.Color("7"))

	panelTitleExpanded = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("10"))

	selLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Width(10)

	selValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11")).
			Italic(true)

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func (m Model) View() string {
	var b strings.Builder

	// Title
	title := titleStyle.Render("LabWC Theme Changer")
	b.WriteString(title + "\n")

	// Current selections (vertical, one per line)
	b.WriteString(m.renderSelections())
	b.WriteString("\n")

	// Panel list (collapsible)
	b.WriteString(m.renderPanels())
	b.WriteString("\n")

	// Help commands (vertical)
	b.WriteString(m.renderHelp())
	b.WriteString("\n")

	// Status line
	status := m.status
	if m.applying {
		status = m.spinner.View() + " " + status
	}
	b.WriteString(statusStyle.Render(status))

	// Wrap everything in a constrained box
	content := b.String()
	return lipgloss.NewStyle().
		Width(m.width).
		MaxWidth(maxWidth).
		Render(content)
}

func (m Model) renderSelections() string {
	var lines []string
	lines = append(lines, dimStyle.Render("─── Current Selection ───"))

	selections := []struct {
		label string
		value string
	}{
		{"GTK", m.selected.GtkTheme},
		{"Icons", m.selected.IconTheme},
		{"LabWC", m.selected.OpenboxTheme},
		{"Kitty", m.selected.KittyTheme},
		{"Wallpaper", m.selected.Wallpaper},
	}

	for _, sel := range selections {
		label := selLabelStyle.Render(sel.label + ":")
		value := selValueStyle.Render(emptyDash(sel.value))
		lines = append(lines, label+value)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderPanels() string {
	var lines []string
	lines = append(lines, dimStyle.Render("─── Theme Panels ───"))

	for t := tabStyle; t < tabCount; t++ {
		// Determine indicator and style
		var indicator string
		var style lipgloss.Style

		isExpanded := m.expanded == t
		isFocused := m.active == t

		if isExpanded {
			indicator = "▼ "
			style = panelTitleExpanded
		} else {
			indicator = "▶ "
			style = panelTitleNormal
		}

		// Add focus marker
		prefix := "  "
		if isFocused && !m.inList {
			prefix = "› "
			style = panelTitleFocused
		}

		// Item count
		count := len(m.lists[t].Items())
		countStr := dimStyle.Render(fmt.Sprintf(" (%d)", count))

		line := prefix + indicator + style.Render(tabNames[t]) + countStr
		lines = append(lines, line)

		// If this panel is expanded, show its list
		if isExpanded {
			listView := m.lists[t].View()
			// Indent the list
			indented := indentLines(listView, "  ")
			lines = append(lines, indented)
		}
	}

	return strings.Join(lines, "\n")
}

var helpKeyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
var helpDescStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))

func (m Model) renderHelp() string {
	var lines []string
	lines = append(lines, dimStyle.Render("─── Commands ───"))

	commands := []struct {
		key  string
		desc string
	}{
		{"↑ ↓", "Navigate panels"},
		{"→ / Enter", "Expand panel"},
		{"← / Esc", "Collapse panel"},
		{"/", "Filter items"},
		{"A", "Apply changes"},
		{"Q", "Quit"},
	}

	for _, cmd := range commands {
		line := helpKeyStyle.Render(fmt.Sprintf("%-10s", cmd.key)) + helpDescStyle.Render(cmd.desc)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func indentLines(s string, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}