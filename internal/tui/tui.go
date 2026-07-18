package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/jacopobonomi/venv-manager/internal/manager"
	"github.com/jacopobonomi/venv-manager/internal/utils"
)

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")).Padding(0, 1)
	panelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Margin(0, 1)
	labelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	valueStyle  = lipgloss.NewStyle().Bold(true)
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575"))
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87"))
)

type keymap struct {
	Refresh key.Binding
	Delete  key.Binding
	Clean   key.Binding
	Info    key.Binding
	Quit    key.Binding
}

func newKeymap() keymap {
	return keymap{
		Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Delete:  key.NewBinding(key.WithKeys("d", "x"), key.WithHelp("d/x", "delete")),
		Clean:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "clean cache")),
		Info:    key.NewBinding(key.WithKeys("i", "enter"), key.WithHelp("i/enter", "load details")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func (k keymap) ShortHelp() []key.Binding {
	return []key.Binding{k.Info, k.Delete, k.Clean, k.Refresh, k.Quit}
}

func (k keymap) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Info, k.Delete, k.Clean, k.Refresh, k.Quit}}
}

type venvItem struct {
	name string
	size int64
}

func (i venvItem) Title() string       { return i.name }
func (i venvItem) Description() string { return utils.FormatSize(i.size) }
func (i venvItem) FilterValue() string { return i.name }

type detailsMsg struct {
	name     string
	packages []string
	size     int64
	err      error
}

type statusMsg struct {
	text string
	err  bool
}

type refreshedMsg struct {
	items []list.Item
	err   error
}

// Model is the TUI's Bubble Tea model.
type Model struct {
	mgr      *manager.Manager
	list     list.Model
	help     help.Model
	keys     keymap
	width    int
	height   int
	status   string
	statusOk bool

	selected string
	details  []string
	loading  bool
}

// New builds a Model backed by the given Manager.
func New(mgr *manager.Manager) Model {
	l := list.New(nil, list.NewDefaultDelegate(), 0, 0)
	l.Title = "🐍 venv-manager"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle

	return Model{
		mgr:      mgr,
		list:     l,
		help:     help.New(),
		keys:     newKeymap(),
		statusOk: true,
	}
}

func (m Model) Init() tea.Cmd {
	return m.refreshCmd()
}

func (m Model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		m.mgr.SetGlobal(true)
		names, err := m.mgr.List()
		if err != nil {
			return refreshedMsg{err: err}
		}
		sizes, _ := m.mgr.GetSize("")
		items := make([]list.Item, 0, len(names))
		for _, n := range names {
			items = append(items, venvItem{name: n, size: sizes[n]})
		}
		return refreshedMsg{items: items}
	}
}

func (m Model) detailsCmd(name string) tea.Cmd {
	return func() tea.Msg {
		pkgs, err := m.mgr.ListPackages(name)
		if err != nil {
			return detailsMsg{name: name, err: err}
		}
		sizes, _ := m.mgr.GetSize(name)
		return detailsMsg{name: name, packages: pkgs, size: sizes[name]}
	}
}

func (m Model) removeCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.mgr.Remove(name); err != nil {
			return statusMsg{text: err.Error(), err: true}
		}
		return statusMsg{text: fmt.Sprintf("removed %q", name)}
	}
}

func (m Model) cleanCmd(name string) tea.Cmd {
	return func() tea.Msg {
		if err := m.mgr.Clean(name); err != nil {
			return statusMsg{text: err.Error(), err: true}
		}
		return statusMsg{text: fmt.Sprintf("cleaned %q", name)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		listW := msg.Width / 3
		if listW < 24 {
			listW = 24
		}
		m.list.SetSize(listW, msg.Height-4)
		return m, nil

	case refreshedMsg:
		if msg.err != nil {
			m.status, m.statusOk = msg.err.Error(), false
			return m, nil
		}
		cmd := m.list.SetItems(msg.items)
		m.status, m.statusOk = fmt.Sprintf("loaded %d venvs", len(msg.items)), true
		return m, cmd

	case detailsMsg:
		m.loading = false
		if msg.err != nil {
			m.status, m.statusOk = msg.err.Error(), false
			return m, nil
		}
		m.selected = msg.name
		m.details = msg.packages
		m.status = fmt.Sprintf("%s — %d packages, %s", msg.name, len(msg.packages), utils.FormatSize(msg.size))
		m.statusOk = true
		return m, nil

	case statusMsg:
		m.status, m.statusOk = msg.text, !msg.err
		return m, m.refreshCmd()

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Refresh):
			m.status = "refreshing…"
			return m, m.refreshCmd()
		case key.Matches(msg, m.keys.Info):
			if it, ok := m.list.SelectedItem().(venvItem); ok {
				m.loading = true
				m.status = "loading packages…"
				return m, m.detailsCmd(it.name)
			}
		case key.Matches(msg, m.keys.Delete):
			if it, ok := m.list.SelectedItem().(venvItem); ok {
				m.status = "removing…"
				return m, m.removeCmd(it.name)
			}
		case key.Matches(msg, m.keys.Clean):
			if it, ok := m.list.SelectedItem().(venvItem); ok {
				m.status = "cleaning…"
				return m, m.cleanCmd(it.name)
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return "loading…"
	}
	left := panelStyle.Width(m.list.Width()).Height(m.height - 4).Render(m.list.View())

	right := m.detailsView()
	rightW := m.width - m.list.Width() - 6
	if rightW < 20 {
		rightW = 20
	}
	rightPanel := panelStyle.Width(rightW).Height(m.height - 4).Render(right)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, rightPanel)

	status := m.status
	if status == "" {
		status = "press ? for help"
	}
	style := statusStyle
	if !m.statusOk {
		style = errStyle
	}
	statusBar := style.Render(status) + "  " + labelStyle.Render(m.help.View(m.keys))
	return lipgloss.JoinVertical(lipgloss.Left, body, statusBar)
}

func (m Model) detailsView() string {
	if m.selected == "" {
		return labelStyle.Render("Select a venv and press ") + valueStyle.Render("enter") +
			labelStyle.Render(" to load packages.")
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(m.selected))
	b.WriteString("\n\n")
	b.WriteString(labelStyle.Render("Packages:\n"))
	if len(m.details) == 0 {
		b.WriteString(labelStyle.Render("  (none installed)"))
		return b.String()
	}
	for _, pkg := range m.details {
		b.WriteString("  • ")
		b.WriteString(pkg)
		b.WriteString("\n")
	}
	return b.String()
}

// Run starts the TUI event loop.
func Run(mgr *manager.Manager) error {
	p := tea.NewProgram(New(mgr), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
