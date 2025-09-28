// internal/ui/app.go
package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cod-e-Codes/tuitar/internal/audio"
	"github.com/Cod-e-Codes/tuitar/internal/models"
	"github.com/Cod-e-Codes/tuitar/internal/storage"
	"github.com/Cod-e-Codes/tuitar/internal/ui/components"
)

type inputMode int

const (
	inputModeNone inputMode = iota
	inputModeSave
	inputModeRename
)

type Model struct {
	state       models.SessionState
	storage     storage.Storage
	tabs        []models.Tab
	audioPlayer *audio.Player

	// Components
	tabEditor  components.TabEditorModel
	tabBrowser components.TabBrowserModel
	statusBar  components.StatusBarModel
	help       help.Model
	textInput  textinput.Model

	// UI State
	windowSize tea.WindowSizeMsg
	showHelp   bool
	inputMode  inputMode
	keys       KeyMap
}

type KeyMap struct {
	Up        key.Binding
	Down      key.Binding
	Left      key.Binding
	Right     key.Binding
	Enter     key.Binding
	Quit      key.Binding
	Help      key.Binding
	Save      key.Binding
	New       key.Binding
	Play      key.Binding
	Insert    key.Binding
	Normal    key.Binding
	Browser   key.Binding
	Delete    key.Binding
	DeleteTab key.Binding
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Save, k.New},
		{k.Insert, k.Normal, k.Browser},
		{k.Play, k.Delete, k.DeleteTab, k.Help, k.Quit},
	}
}

func NewKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("h", "left"),
			key.WithHelp("←/h", "move left"),
		),
		Right: key.NewBinding(
			key.WithKeys("l", "right"),
			key.WithHelp("→/l", "move right"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select/confirm"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Save: key.NewBinding(
			key.WithKeys("ctrl+s"),
			key.WithHelp("ctrl+s", "save tab"),
		),
		New: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "new tab"),
		),
		Play: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "play/pause"),
		),
		Insert: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "insert mode"),
		),
		Normal: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "normal mode"),
		),
		Browser: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "browser/editor"),
		),
		Delete: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete fret"),
		),
		DeleteTab: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete tab"),
		),
	}
}

func NewModel(storage storage.Storage) Model {
	tabs, _ := storage.LoadAllTabs()

	textInput := textinput.New()
	textInput.Placeholder = "Enter tab name..."
	textInput.Focus()

	m := Model{
		storage:     storage,
		tabs:        tabs,
		keys:        NewKeyMap(),
		help:        help.New(),
		tabBrowser:  components.NewTabBrowser(tabs),
		statusBar:   components.NewStatusBar(),
		textInput:   textInput,
		audioPlayer: audio.NewPlayer(),
	}

	m.state.ViewMode = models.ViewBrowser
	m.state.EditMode = models.EditNormal

	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.SetWindowTitle("Tuitar - Guitar Tab TUI"),
		m.tick(),
	)
}

// tick returns a command that sends a tick message every 100ms for playback updates
func (m Model) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

type tickMsg struct{}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tickMsg:
		// Update playback highlights if playing
		if m.audioPlayer.IsPlaying() && m.state.ViewMode == models.ViewEditor {
			highlighted := m.audioPlayer.GetHighlighted()
			m.tabEditor.SetHighlightedPositions(highlighted)
		}
		return m, m.tick()

	case tea.WindowSizeMsg:
		m.windowSize = msg
		m.tabEditor.SetSize(msg.Width, msg.Height-3)
		m.tabBrowser.SetSize(msg.Width, msg.Height-3)

	case tea.KeyMsg:
		// Handle input mode first
		if m.inputMode != inputModeNone {
			return m.updateInput(msg)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			if m.audioPlayer.IsPlaying() {
				m.audioPlayer.Stop()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp
			return m, nil

		case key.Matches(msg, m.keys.Browser):
			// Toggle between browser and editor
			if m.state.ViewMode == models.ViewBrowser {
				if m.state.CurrentTab != nil {
					m.state.ViewMode = models.ViewEditor
				}
			} else {
				m.state.ViewMode = models.ViewBrowser
			}
			return m, nil

		case key.Matches(msg, m.keys.New):
			newTab := models.NewTestTab("New Tab")
			m.state.CurrentTab = newTab
			m.tabEditor = components.NewTabEditor(newTab)
			m.tabEditor.SetEditMode(models.EditNormal)
			m.state.ViewMode = models.ViewEditor
			m.state.EditMode = models.EditNormal
			m.statusBar.SetStatus("Created new tab with test notes")
			return m, nil

		case key.Matches(msg, m.keys.Save):
			if m.state.CurrentTab != nil {
				if m.state.CurrentTab.ID == 0 || m.state.CurrentTab.Name == "New Tab" {
					// New tab or default name - prompt for name
					m.inputMode = inputModeSave
					m.textInput.SetValue(m.state.CurrentTab.Name)
					m.textInput.Focus()
				} else {
					// Existing tab - save directly
					m.saveCurrentTab()
				}
			}
			return m, nil

		case key.Matches(msg, m.keys.Play):
			if m.state.ViewMode == models.ViewEditor && m.state.CurrentTab != nil {
				if m.audioPlayer.IsPlaying() {
					m.audioPlayer.Stop()
					m.statusBar.SetStatus("Playback stopped")
				} else {
					err := m.audioPlayer.PlayTab(m.state.CurrentTab)
					if err != nil {
						m.statusBar.SetStatus("Playback error: " + err.Error())
					} else {
						m.statusBar.SetStatus("Playing tab...")
					}
				}
			}
			return m, nil
		}

		// Handle view-specific key presses
		switch m.state.ViewMode {
		case models.ViewBrowser:
			return m.updateBrowser(msg)
		case models.ViewEditor:
			return m.updateEditor(msg)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.inputMode = inputModeNone
		m.textInput.Blur()
		return m, nil

	case tea.KeyEnter:
		value := m.textInput.Value()
		if value != "" {
			if m.inputMode == inputModeSave {
				m.state.CurrentTab.Name = value
				m.saveCurrentTab()
			}
		}
		m.inputMode = inputModeNone
		m.textInput.Blur()
		m.textInput.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *Model) saveCurrentTab() {
	err := m.storage.SaveTab(m.state.CurrentTab)
	if err != nil {
		m.statusBar.SetStatus("Error saving tab: " + err.Error())
	} else {
		m.statusBar.SetStatus("Tab saved: " + m.state.CurrentTab.Name)
		// Refresh tabs list
		if tabs, err := m.storage.LoadAllTabs(); err == nil {
			m.tabs = tabs
			m.tabBrowser.SetTabs(tabs)
		}
	}
}

func (m Model) updateBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Enter):
		if len(m.tabs) > 0 && m.tabBrowser.Cursor() < len(m.tabs) {
			selectedTab := &m.tabs[m.tabBrowser.Cursor()]
			tabCopy := *selectedTab
			m.state.CurrentTab = &tabCopy
			m.tabEditor = components.NewTabEditor(&tabCopy)
			m.tabEditor.SetEditMode(models.EditNormal)
			m.state.ViewMode = models.ViewEditor
			m.state.EditMode = models.EditNormal
			m.statusBar.SetStatus("Editing: " + tabCopy.Name)
		}
		return m, nil

	case key.Matches(msg, m.keys.DeleteTab):
		if len(m.tabs) > 0 && m.tabBrowser.Cursor() < len(m.tabs) {
			selectedTab := &m.tabs[m.tabBrowser.Cursor()]
			err := m.storage.DeleteTab(selectedTab.ID)
			if err != nil {
				m.statusBar.SetStatus("Error deleting tab: " + err.Error())
			} else {
				m.statusBar.SetStatus("Deleted tab: " + selectedTab.Name)
				// Refresh tabs list
				if tabs, err := m.storage.LoadAllTabs(); err == nil {
					m.tabs = tabs
					m.tabBrowser.SetTabs(tabs)
					// Adjust cursor if it's now out of bounds
					if m.tabBrowser.Cursor() >= len(m.tabs) && len(m.tabs) > 0 {
						m.tabBrowser.SetCursor(len(m.tabs) - 1)
					}
				}
			}
		}
		return m, nil
	}

	m.tabBrowser, cmd = m.tabBrowser.Update(msg)
	return m, cmd
}

func (m Model) updateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Insert):
		m.state.EditMode = models.EditInsert
		m.tabEditor.SetEditMode(models.EditInsert)
		m.statusBar.SetStatus("-- INSERT MODE --")
		return m, nil

	case key.Matches(msg, m.keys.Normal):
		m.state.EditMode = models.EditNormal
		m.tabEditor.SetEditMode(models.EditNormal)
		m.statusBar.SetStatus("-- NORMAL MODE --")
		return m, nil
	}

	// Pass the message to the tab editor
	m.tabEditor, cmd = m.tabEditor.Update(msg)

	// Update the current tab if it has changed
	if m.tabEditor.HasChanged() {
		m.state.CurrentTab = m.tabEditor.GetTab()
	}

	return m, cmd
}

func (m Model) View() string {
	// Handle input dialogs
	if m.inputMode != inputModeNone {
		return m.renderInputDialog()
	}

	if m.showHelp {
		return m.renderHelp()
	}

	var content string
	switch m.state.ViewMode {
	case models.ViewBrowser:
		content = m.renderBrowser()
	case models.ViewEditor:
		content = m.renderEditor()
	}

	statusBar := m.statusBar.View()
	return lipgloss.JoinVertical(lipgloss.Left, content, statusBar)
}

func (m Model) renderInputDialog() string {
	var title string
	switch m.inputMode {
	case inputModeSave:
		title = "Save Tab As:"
	case inputModeRename:
		title = "Rename Tab:"
	}

	dialog := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1, 2).
		Width(50).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Render(title),
			"",
			m.textInput.View(),
			"",
			lipgloss.NewStyle().Faint(true).Render("Enter: Save • Esc: Cancel"),
		))

	return lipgloss.Place(m.windowSize.Width, m.windowSize.Height,
		lipgloss.Center, lipgloss.Center, dialog)
}

func (m Model) renderHelp() string {
	helpContent := lipgloss.NewStyle().
		Padding(1).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")).Render("Tuitar - Guitar Tab Editor Help"),
			"",
			lipgloss.NewStyle().Bold(true).Render("Global Keys:"),
			"  q, Ctrl+C     - Quit application",
			"  ?             - Toggle this help",
			"  Ctrl+N        - Create new tab",
			"  Ctrl+S        - Save current tab",
			"  Tab           - Switch between browser and editor",
			"",
			lipgloss.NewStyle().Bold(true).Render("Browser Mode:"),
			"  ↑/k, ↓/j      - Navigate tab list",
			"  Enter         - Edit selected tab",
			"  d             - Delete selected tab",
			"",
			lipgloss.NewStyle().Bold(true).Render("Editor Mode - Normal:"),
			"  ↑/k, ↓/j      - Move between strings",
			"  ←/h, →/l      - Move along string",
			"  w/b           - Move to next/previous measure",
			"  g/$           - Move to start/end of measure",
			"  PgUp/PgDn     - Page up/down scrolling",
			"  i             - Enter insert mode",
			"  x             - Delete fret (replace with -)",
			"  Space         - Play/pause tab",
			"  m             - Add new measure",
			"  M             - Remove last measure",
			"",
			lipgloss.NewStyle().Bold(true).Render("Editor Mode - Insert:"),
			"  0-9           - Insert fret number (auto-advance)",
			"  -             - Insert rest (auto-advance)",
			"  Backspace     - Delete and move back",
			"  Esc           - Return to normal mode",
			"  Arrow keys    - Navigate",
			"",
			lipgloss.NewStyle().Faint(true).Render("Press ? again to close this help"),
		))

	return helpContent
}

func (m Model) renderBrowser() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Render("Tuitar - Guitar Tab Browser")

	help := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render("Enter: Edit • Ctrl+N: New • d: Delete • Tab: Editor • ?: Help • Q: Quit")

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.tabBrowser.View(),
		"",
		help,
	)
}

func (m Model) renderEditor() string {
	if m.state.CurrentTab == nil {
		return "No tab selected"
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")).
		Render(fmt.Sprintf("Editing: %s", m.state.CurrentTab.Name))

	// Show playback status
	playStatus := ""
	if m.audioPlayer.IsPlaying() {
		playStatus = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")).
			Render(" [PLAYING]")
	}

	mode := "NORMAL"
	modeColor := lipgloss.Color("12")
	if m.state.EditMode == models.EditInsert {
		mode = "INSERT"
		modeColor = lipgloss.Color("11")
	}

	modeIndicator := lipgloss.NewStyle().
		Bold(true).
		Foreground(modeColor).
		Render(fmt.Sprintf("-- %s --", mode)) + playStatus

	var help string
	if m.state.EditMode == models.EditInsert {
		help = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("0-9: Insert fret • -: Rest • Esc: Normal • Arrows: Navigate • Backspace: Delete back")
	} else {
		help = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("I: Insert • X: Delete • Space: Play • m: Add Measure • Ctrl+S: Save • Tab: Browser • Arrows: Navigate")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		m.tabEditor.View(),
		"",
		modeIndicator,
		help,
	)
}
