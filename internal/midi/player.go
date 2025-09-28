// internal/midi/player.go
package midi

import (
	"strconv"
	"sync"
	"time"

	"github.com/Cod-e-Codes/tuitar/internal/models"
)

type Player struct {
	mu           sync.RWMutex
	isPlaying    bool
	position     int
	tempo        int
	notes        []PlayableNote
	highlighted  []models.Position
	stopChan     chan bool
	currentTab   *models.Tab
	playbackTime time.Duration
}

type PlayableNote struct {
	MidiNote int
	Start    time.Duration
	Duration time.Duration
	Velocity int
	String   int
	Position int
}

func NewPlayer() *Player {
	return &Player{
		tempo:    120,
		stopChan: make(chan bool, 1),
	}
}

func (p *Player) PlayTab(tab *models.Tab) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isPlaying {
		return nil
	}

	p.currentTab = tab
	p.notes = p.convertTabToNotes(tab)
	p.isPlaying = true
	p.position = 0
	p.playbackTime = 0

	// Clear the stop channel
	select {
	case <-p.stopChan:
	default:
	}

	go p.playbackLoop()

	return nil
}

func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.isPlaying {
		p.isPlaying = false
		select {
		case p.stopChan <- true:
		default:
		}
		p.highlighted = nil
		p.position = 0
		p.playbackTime = 0
	}
}

func (p *Player) IsPlaying() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.isPlaying
}

func (p *Player) GetHighlighted() []models.Position {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]models.Position, len(p.highlighted))
	copy(result, p.highlighted)
	return result
}

func (p *Player) GetPosition() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.position
}

func (p *Player) convertTabToNotes(tab *models.Tab) []PlayableNote {
	var notes []PlayableNote

	// Standard guitar tuning MIDI notes (low to high)
	// E(6th) A(5th) D(4th) G(3rd) B(2nd) e(1st) - but our array is reversed
	stringMidiNotes := [6]int{64, 59, 55, 50, 45, 40} // e B G D A E (high to low as displayed)

	maxLength := 0
	for _, line := range tab.Content {
		if len(line) > maxLength {
			maxLength = len(line)
		}
	}

	// Use the tab's tempo if available, otherwise default
	tempo := tab.Tempo
	if tempo <= 0 {
		tempo = 120
	}

	// Calculate note duration based on tempo (assume 16th notes)
	beatDuration := time.Minute / time.Duration(tempo*4)

	for pos := 0; pos < maxLength; pos++ {
		for stringIdx, line := range tab.Content {
			if pos < len(line) && line[pos] != '-' && line[pos] != '|' && line[pos] != ' ' {
				if fret, err := strconv.Atoi(string(line[pos])); err == nil && fret >= 0 && fret <= 24 {
					midiNote := stringMidiNotes[stringIdx] + fret

					note := PlayableNote{
						MidiNote: midiNote,
						Start:    time.Duration(pos) * beatDuration,
						Duration: beatDuration * 3 / 4, // Note length (slightly shorter than beat)
						Velocity: 127,
						String:   stringIdx,
						Position: pos,
					}
					notes = append(notes, note)
				}
			}
		}
	}

	return notes
}

func (p *Player) playbackLoop() {
	defer func() {
		p.mu.Lock()
		p.isPlaying = false
		p.highlighted = nil
		p.position = 0
		p.playbackTime = 0
		p.mu.Unlock()
	}()

	// Use the tab's tempo
	tempo := 120
	if p.currentTab != nil && p.currentTab.Tempo > 0 {
		tempo = p.currentTab.Tempo
	}

	beatDuration := time.Minute / time.Duration(tempo*4) // 16th notes
	ticker := time.NewTicker(beatDuration)
	defer ticker.Stop()

	maxPos := 0
	for _, note := range p.notes {
		if note.Position > maxPos {
			maxPos = note.Position
		}
	}

	// If no notes, determine max position from tab content
	if maxPos == 0 && p.currentTab != nil {
		for _, line := range p.currentTab.Content {
			if len(line) > maxPos {
				maxPos = len(line)
			}
		}
	}

	startTime := time.Now()

	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.mu.Lock()

			// Update playback time
			p.playbackTime = time.Since(startTime)

			// Update highlighted positions based on current position
			p.highlighted = nil
			for _, note := range p.notes {
				if note.Position == p.position {
					p.highlighted = append(p.highlighted, models.Position{
						String:   note.String,
						Position: note.Position,
					})
				}
			}

			// Simulate playing notes at this position
			// In a real implementation, this would trigger actual MIDI output
			notesAtPosition := 0
			for _, note := range p.notes {
				if note.Position == p.position {
					notesAtPosition++
					// Here you would send MIDI note on/off commands
					// fmt.Printf("Playing MIDI note %d on string %d at position %d\n",
					//     note.MidiNote, note.String, note.Position)
				}
			}

			p.position++

			// Check if we've reached the end
			if p.position > maxPos {
				p.mu.Unlock()
				return
			}

			p.mu.Unlock()
		}
	}
}

func (p *Player) GetPlaybackInfo() (position int, totalLength int, isPlaying bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	maxPos := 0
	if p.currentTab != nil {
		for _, line := range p.currentTab.Content {
			if len(line) > maxPos {
				maxPos = len(line)
			}
		}
	}

	return p.position, maxPos, p.isPlaying
}

// SetTempo allows changing playback tempo
func (p *Player) SetTempo(tempo int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if tempo > 0 && tempo <= 300 {
		p.tempo = tempo
		if p.currentTab != nil {
			p.currentTab.Tempo = tempo
		}
	}
}
