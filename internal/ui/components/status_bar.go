// internal/ui/components/status_bar.go
package components

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type StatusBarModel struct {
	message   string
	timestamp time.Time
}

func NewStatusBar() StatusBarModel {
	return StatusBarModel{}
}

func (m *StatusBarModel) SetStatus(message string) {
	m.message = message
	m.timestamp = time.Now()
}

func (m StatusBarModel) View() string {
	if m.message == "" {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("8")).
			Foreground(lipgloss.Color("15")).
			Width(80).
			Render(" tuitar - Ready")
	}
	
	// Show message for 3 seconds, then clear
	if time.Since(m.timestamp) > 3*time.Second {
		return lipgloss.NewStyle().
			Background(lipgloss.Color("8")).
			Foreground(lipgloss.Color("15")).
			Width(80).
			Render(" Guitar Tab TUI - Ready")
	}
	
	return lipgloss.NewStyle().
		Background(lipgloss.Color("11")).
		Foreground(lipgloss.Color("0")).
		Width(80).
		Render(fmt.Sprintf(" %s", m.message))
}
