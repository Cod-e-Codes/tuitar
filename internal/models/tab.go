// internal/models/tab.go
package models

import (
	"time"
)

type Tab struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Artist        string    `json:"artist" db:"artist"`
	Content       [6]string `json:"content" db:"content"` // 6 strings - now supports variable length
	Tuning        [6]string `json:"tuning" db:"tuning"`   // E A D G B e
	Tempo         int       `json:"tempo" db:"tempo"`
	TimeSignature string    `json:"time_signature" db:"time_signature"`
	Measures      int       `json:"measures" db:"measures"` // Number of measures
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

func NewEmptyTab(name string) *Tab {
	// Start with 4 measures, each 16 characters long (64 total)
	emptyLine := "----------------" + "----------------" + "----------------" + "----------------"
	return &Tab{
		Name:          name,
		Artist:        "",
		Content:       [6]string{emptyLine, emptyLine, emptyLine, emptyLine, emptyLine, emptyLine},
		Tuning:        [6]string{"e", "B", "G", "D", "A", "E"},
		Tempo:         120,
		TimeSignature: "4/4",
		Measures:      4,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// NewTestTab creates a tab with some sample notes for testing
func NewTestTab(name string) *Tab {
	// Create a simple test tab with some notes
	tab := NewEmptyTab(name)

	// Add some test notes on the first string (high E)
	content := []rune(tab.Content[0])
	content[0] = '0'  // Open string
	content[4] = '2'  // 2nd fret
	content[8] = '4'  // 4th fret
	content[12] = '5' // 5th fret
	tab.Content[0] = string(content)

	// Add some notes on the second string (B)
	content = []rune(tab.Content[1])
	content[2] = '1'  // 1st fret
	content[6] = '3'  // 3rd fret
	content[10] = '5' // 5th fret
	tab.Content[1] = string(content)

	return tab
}

type Position struct {
	String   int
	Position int
}

type ViewMode int

const (
	ViewHome ViewMode = iota
	ViewEditor
	ViewBrowser
	ViewSettings
	ViewHelp
)

type EditMode int

const (
	EditNormal EditMode = iota
	EditInsert
	EditSelect
)

type PlaybackState struct {
	IsPlaying   bool
	Position    int
	Highlighted []Position
	Tempo       int
}

type SessionState struct {
	CurrentTab    *Tab
	CursorPos     Position
	ViewMode      ViewMode
	PlaybackState PlaybackState
	EditMode      EditMode
}

// Helper methods for Tab
const MeasureLength = 16 // Characters per measure

// AddMeasure adds a new measure to the tab
func (t *Tab) AddMeasure() {
	measureLine := "----------------"
	for i := 0; i < 6; i++ {
		t.Content[i] += measureLine
	}
	t.Measures++
	t.UpdatedAt = time.Now()
}

// RemoveMeasure removes the last measure from the tab
func (t *Tab) RemoveMeasure() {
	if t.Measures <= 1 {
		return // Keep at least one measure
	}

	for i := 0; i < 6; i++ {
		if len(t.Content[i]) >= MeasureLength {
			t.Content[i] = t.Content[i][:len(t.Content[i])-MeasureLength]
		}
	}
	t.Measures--
	t.UpdatedAt = time.Now()
}

// GetMeasureCount returns the number of measures based on content length
func (t *Tab) GetMeasureCount() int {
	if len(t.Content[0]) == 0 {
		return 0
	}
	return len(t.Content[0]) / MeasureLength
}

// GetTotalLength returns the total number of positions in the tab
func (t *Tab) GetTotalLength() int {
	if len(t.Content) == 0 {
		return 0
	}
	return len(t.Content[0])
}
