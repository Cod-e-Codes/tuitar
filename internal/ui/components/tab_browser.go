// internal/ui/components/tab_browser.go
package components

import (
	"fmt"
	"strings"

	"github.com/Cod-e-Codes/tuitar/internal/models"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TabBrowserModel struct {
	tabs     []models.Tab
	cursor   int
	viewport viewport.Model
	width    int
	height   int
}

func NewTabBrowser(tabs []models.Tab) TabBrowserModel {
	vp := viewport.New(80, 20)

	return TabBrowserModel{
		tabs:     tabs,
		viewport: vp,
	}
}

func (m *TabBrowserModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}

func (m TabBrowserModel) Update(msg tea.Msg) (TabBrowserModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "j", "down":
			if m.cursor < len(m.tabs)-1 {
				m.cursor++
			}
		case "home":
			m.cursor = 0
		case "end":
			m.cursor = len(m.tabs) - 1
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m TabBrowserModel) View() string {
	if len(m.tabs) == 0 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("8")).
			Render("No tabs found. Press Ctrl+N to create a new tab.")
	}

	var items []string

	for i, tab := range m.tabs {
		style := lipgloss.NewStyle()

		if i == m.cursor {
			style = style.Background(lipgloss.Color("12")).Foreground(lipgloss.Color("15"))
		}

		// Format: [ID] Name - Artist (Date)
		item := fmt.Sprintf("[%d] %s", tab.ID, tab.Name)
		if tab.Artist != "" {
			item += fmt.Sprintf(" - %s", tab.Artist)
		}
		item += fmt.Sprintf(" (%s)", tab.UpdatedAt.Format("2006-01-02"))

		items = append(items, style.Render(item))
	}

	content := strings.Join(items, "\n")
	m.viewport.SetContent(content)

	return m.viewport.View()
}

func (m TabBrowserModel) Cursor() int {
	return m.cursor
}

func (m *TabBrowserModel) SetTabs(tabs []models.Tab) {
	m.tabs = tabs
	if m.cursor >= len(tabs) {
		m.cursor = len(tabs) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m *TabBrowserModel) SetCursor(cursor int) {
	if cursor >= 0 && cursor < len(m.tabs) {
		m.cursor = cursor
	}
}
