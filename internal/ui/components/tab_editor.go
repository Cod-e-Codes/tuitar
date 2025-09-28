// internal/ui/components/tab_editor.go
package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Cod-e-Codes/tuitar/internal/models"
)

type HighlightUpdateMsg struct {
	Positions []models.Position
}

type TabEditorModel struct {
	tab            *models.Tab
	cursor         models.Position
	viewport       viewport.Model
	width          int
	height         int
	changed        bool
	editMode       models.EditMode
	highlightedPos []models.Position // For playback highlighting
	showHelp       bool              // Show measure management help
}

func NewTabEditor(tab *models.Tab) TabEditorModel {
	vp := viewport.New(80, 20)

	// Initialize tab content if it's empty
	if tab.Content[0] == "" {
		emptyLine := "----------------" + "----------------" + "----------------" + "----------------"
		tab.Content = [6]string{emptyLine, emptyLine, emptyLine, emptyLine, emptyLine, emptyLine}
		tab.Measures = 4
	}

	return TabEditorModel{
		tab:      tab,
		viewport: vp,
		cursor:   models.Position{String: 0, Position: 0},
		editMode: models.EditNormal,
	}
}

func (m *TabEditorModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height - 4 // Reserve space for headers
}

// Updated to set changed = true to force re-render on highlight change
func (m *TabEditorModel) SetHighlightedPositions(positions []models.Position) {
	m.highlightedPos = positions
	m.changed = true
}

// Update now handles external highlight update message to refresh highlights
func (m TabEditorModel) Update(msg tea.Msg) (TabEditorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case HighlightUpdateMsg:
		m.highlightedPos = msg.Positions
		m.changed = true
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		// Navigation keys work in both modes
		case "h", "left":
			if m.cursor.Position > 0 {
				m.cursor.Position--
			}
		case "l", "right":
			maxPos := len(m.tab.Content[m.cursor.String]) - 1
			if m.cursor.Position < maxPos {
				m.cursor.Position++
			}
		case "k", "up":
			if m.cursor.String > 0 {
				m.cursor.String--
			}
		case "j", "down":
			if m.cursor.String < 5 {
				m.cursor.String++
			}

		// Page scrolling
		case "pgup":
			// Scroll up by viewport height
			for i := 0; i < m.viewport.Height; i++ {
				m.viewport.ScrollUp(1)
			}
		case "pgdown":
			// Scroll down by viewport height
			for i := 0; i < m.viewport.Height; i++ {
				m.viewport.ScrollDown(1)
			}

		// More intuitive cursor movement
		case "w":
			// Move to next word/measure boundary (forward)
			if m.editMode == models.EditNormal {
				// Move to next measure boundary
				nextMeasurePos := ((m.cursor.Position / models.MeasureLength) + 1) * models.MeasureLength
				maxPos := len(m.tab.Content[m.cursor.String]) - 1
				if nextMeasurePos <= maxPos {
					m.cursor.Position = nextMeasurePos
				} else {
					m.cursor.Position = maxPos
				}
			}
		case "b":
			// Move to previous word/measure boundary (backward)
			if m.editMode == models.EditNormal {
				// Move to previous measure boundary
				if m.cursor.Position > 0 {
					prevMeasurePos := ((m.cursor.Position - 1) / models.MeasureLength) * models.MeasureLength
					m.cursor.Position = prevMeasurePos
				}
			}
		case "g":
			// Move to beginning of current measure (like 'gg' in vim)
			if m.editMode == models.EditNormal {
				measureStart := (m.cursor.Position / models.MeasureLength) * models.MeasureLength
				m.cursor.Position = measureStart
			}
		case "$":
			// Move to end of current measure
			if m.editMode == models.EditNormal {
				measureEnd := ((m.cursor.Position/models.MeasureLength)+1)*models.MeasureLength - 1
				maxPos := len(m.tab.Content[m.cursor.String]) - 1
				if measureEnd <= maxPos {
					m.cursor.Position = measureEnd
				} else {
					m.cursor.Position = maxPos
				}
			}
		case "home":
			m.cursor.Position = 0
		case "end":
			m.cursor.Position = len(m.tab.Content[m.cursor.String]) - 1

		// Insert mode specific keys
		case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
			if m.editMode == models.EditInsert {
				m.insertCharAt(m.cursor, rune(msg.String()[0]))
				m.changed = true
				if m.cursor.Position < len(m.tab.Content[m.cursor.String])-1 {
					m.cursor.Position++
				}
			}
		case "-":
			if m.editMode == models.EditInsert {
				m.insertCharAt(m.cursor, '-')
				m.changed = true
				if m.cursor.Position < len(m.tab.Content[m.cursor.String])-1 {
					m.cursor.Position++
				}
			}

		// Delete key works in normal mode
		case "x":
			if m.editMode == models.EditNormal {
				m.deleteCharAt(m.cursor)
				m.changed = true
			}

		// Backspace works in insert mode
		case "backspace", "ctrl+h":
			if m.editMode == models.EditInsert && m.cursor.Position > 0 {
				m.cursor.Position--
				m.deleteCharAt(m.cursor)
				m.changed = true
			}

		// Measure management keys (work in normal mode)
		// Use 'm' for add measure and 'M' for remove measure (simpler than Ctrl+M which conflicts with Enter)
		case "m":
			if m.editMode == models.EditNormal {
				m.tab.AddMeasure()
				m.changed = true
			}
		case "M":
			if m.editMode == models.EditNormal {
				m.tab.RemoveMeasure()
				// Adjust cursor position if it's beyond the new length
				maxPos := m.tab.GetTotalLength() - 1
				if m.cursor.Position > maxPos {
					m.cursor.Position = maxPos
				}
				m.changed = true
			}
		case "?":
			if m.editMode == models.EditNormal {
				m.showHelp = !m.showHelp
				m.changed = true
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *TabEditorModel) insertCharAt(pos models.Position, char rune) {
	line := []rune(m.tab.Content[pos.String])
	if pos.Position < len(line) {
		line[pos.Position] = char
		m.tab.Content[pos.String] = string(line)
	}
}

func (m *TabEditorModel) deleteCharAt(pos models.Position) {
	line := []rune(m.tab.Content[pos.String])
	if pos.Position < len(line) {
		line[pos.Position] = '-'
		m.tab.Content[pos.String] = string(line)
	}
}

func (m TabEditorModel) View() string {
	var lines []string

	// String labels (high to low pitch, matching guitar orientation)
	stringLabels := []string{"e", "B", "G", "D", "A", "E"}

	// Helper to check if position is highlighted
	isHighlighted := func(str, pos int) bool {
		for _, hp := range m.highlightedPos {
			if hp.String == str && hp.Position == pos {
				return true
			}
		}
		return false
	}

	// Helper to check if position is at a measure boundary
	isMeasureBoundary := func(pos int) bool {
		return pos > 0 && pos%models.MeasureLength == 0
	}

	// Calculate how many measures can fit on one line
	// Use a reasonable default width if width is 0 (not set yet)
	displayWidth := m.width
	if displayWidth == 0 {
		displayWidth = 120 // Default terminal width
	}

	// Account for string labels (3 chars) + pipe (1 char) + pipe at end (1 char) = 5 chars
	availableWidth := displayWidth - 5
	measuresPerLine := availableWidth / (models.MeasureLength + 1) // +1 for spacing between measures
	if measuresPerLine < 1 {
		measuresPerLine = 1
	}

	// Debug: let's be more generous with side-by-side display
	// Try to fit at least 2-3 measures side by side if possible
	if availableWidth >= (models.MeasureLength*2)+2 {
		measuresPerLine = 2
	}
	if availableWidth >= (models.MeasureLength*3)+3 {
		measuresPerLine = 3
	}
	if availableWidth >= (models.MeasureLength*4)+4 {
		measuresPerLine = 4
	}

	// Render measures in blocks
	for measureStart := 0; measureStart < m.tab.GetMeasureCount(); measureStart += measuresPerLine {
		// Add spacing between measure blocks (except for the first one)
		if measureStart > 0 {
			lines = append(lines, "")
		}

		// Determine how many measures to render in this block
		measuresInBlock := measuresPerLine
		if measureStart+measuresInBlock > m.tab.GetMeasureCount() {
			measuresInBlock = m.tab.GetMeasureCount() - measureStart
		}

		// Render each string for this block of measures
		for i, label := range stringLabels {
			line := lipgloss.NewStyle().
				Foreground(lipgloss.Color("14")).
				Render(label + "|")

			// Render each measure in this block
			for measureIdx := 0; measureIdx < measuresInBlock; measureIdx++ {
				actualMeasureIdx := measureStart + measureIdx
				measureStartPos := actualMeasureIdx * models.MeasureLength
				measureEndPos := measureStartPos + models.MeasureLength

				// Get the content for this measure
				content := m.tab.Content[i]
				if measureEndPos > len(content) {
					measureEndPos = len(content)
				}

				// Render this measure
				for pos := measureStartPos; pos < measureEndPos; pos++ {
					if pos >= len(content) {
						line += "-" // Fill with dashes if content is shorter
						continue
					}

					char := content[pos]
					style := lipgloss.NewStyle()

					// Use if-else instead of switch to avoid gocritic warning
					if m.cursor.String == i && m.cursor.Position == pos {
						// Highlight cursor position (takes precedence)
						if m.editMode == models.EditInsert {
							style = style.Background(lipgloss.Color("11")).Foreground(lipgloss.Color("0"))
						} else {
							style = style.Background(lipgloss.Color("12")).Foreground(lipgloss.Color("15"))
						}
					} else if isHighlighted(i, pos) {
						// Highlight playback positions with cyan background
						style = style.Background(lipgloss.Color("37")).Foreground(lipgloss.Color("0"))
					} else if isMeasureBoundary(pos % models.MeasureLength) {
						// Add subtle highlighting for measure boundaries
						style = style.Foreground(lipgloss.Color("8"))
					}

					line += style.Render(string(char))
				}

				// Add spacing between measures (except for the last one in the block)
				if measureIdx < measuresInBlock-1 {
					line += " "
				}
			}

			line += lipgloss.NewStyle().
				Foreground(lipgloss.Color("14")).
				Render("|")

			lines = append(lines, line)
		}

		// Add measure numbers below this block
		measureLine := "   "
		for measureIdx := 0; measureIdx < measuresInBlock; measureIdx++ {
			actualMeasureIdx := measureStart + measureIdx
			measureNum := fmt.Sprintf("%d", actualMeasureIdx+1)
			// Center the measure number under each measure
			padding := (models.MeasureLength - len(measureNum)) / 2
			measureLine += strings.Repeat(" ", padding) + measureNum + strings.Repeat(" ", models.MeasureLength-padding-len(measureNum))

			// Add spacing between measure numbers (except for the last one)
			if measureIdx < measuresInBlock-1 {
				measureLine += " "
			}
		}
		lines = append(lines, measureLine)
	}

	// Add help information if requested
	if m.showHelp {
		helpLines := []string{
			"",
			"Measure Management:",
			"  m                   - Add a new measure",
			"  M                   - Remove last measure",
			"  ?                   - Toggle this help",
			"",
			"Navigation:",
			"  h/j/k/l    - Move cursor (left/down/up/right)",
			"  w/b        - Move to next/previous measure",
			"  g/$        - Move to start/end of measure",
			"  Home/End   - Move to start/end of string",
			"  PgUp/PgDn  - Page up/down scrolling",
			"",
			"Editing:",
			"  i          - Enter insert mode",
			"  Esc        - Exit insert mode",
			"  x          - Delete character (normal mode)",
			"  Backspace  - Delete character (insert mode)",
		}
		lines = append(lines, helpLines...)
	}

	content := strings.Join(lines, "\n")
	m.viewport.SetContent(content)

	// Update viewport to follow cursor if needed
	m.updateViewportForCursor()

	return m.viewport.View()
}

func (m TabEditorModel) HasChanged() bool {
	return m.changed
}

func (m *TabEditorModel) updateViewportForCursor() {
	// Calculate which line the cursor is on
	cursorLine := m.cursor.String + (m.cursor.Position/models.MeasureLength)*7 // 6 strings + 1 measure number line

	// If cursor is below visible area, scroll down
	if cursorLine >= m.viewport.YOffset+m.viewport.Height {
		m.viewport.ScrollDown(1)
	}

	// If cursor is above visible area, scroll up
	if cursorLine < m.viewport.YOffset {
		m.viewport.ScrollUp(1)
	}
}

func (m *TabEditorModel) ResetChanged() {
	m.changed = false
}

func (m TabEditorModel) GetTab() *models.Tab {
	return m.tab
}

func (m *TabEditorModel) SetEditMode(mode models.EditMode) {
	m.editMode = mode
}

func (m TabEditorModel) GetEditMode() models.EditMode {
	return m.editMode
}

func (m TabEditorModel) GetCursor() models.Position {
	return m.cursor
}
