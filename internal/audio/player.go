//go:build cgo
// +build cgo

// internal/audio/player.go
package audio

import (
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"github.com/Cod-e-Codes/tuitar/internal/models"
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/generators"
	"github.com/faiface/beep/speaker"
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
	sampleRate   beep.SampleRate
	mixer        *beep.Mixer
	ctrl         *beep.Ctrl
}

type PlayableNote struct {
	Frequency float64
	Start     time.Duration
	Duration  time.Duration
	Volume    float64
	String    int
	Position  int
}

func NewPlayer() *Player {
	sampleRate := beep.SampleRate(44100)

	// Initialize the speaker
	err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
	if err != nil {
		fmt.Printf("Failed to initialize speaker: %v\n", err)
	}

	mixer := &beep.Mixer{}
	
	// Create a control wrapper for the mixer
	ctrl := &beep.Ctrl{Streamer: mixer, Paused: true}
	
	// Play the mixer through the speaker
	speaker.Play(ctrl)

	return &Player{
		tempo:      120,
		stopChan:   make(chan bool, 1),
		sampleRate: sampleRate,
		mixer:      mixer,
		ctrl:       ctrl,
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
	fmt.Printf("Converted tab to %d notes\n", len(p.notes))

	if len(p.notes) == 0 {
		return fmt.Errorf("no playable notes found in tab")
	}

	p.isPlaying = true
	p.position = 0
	p.playbackTime = 0

	// Clear the stop channel
	select {
	case <-p.stopChan:
	default:
	}

	// Clear the mixer and unpause playback
	p.mixer.Clear()
	speaker.Lock()
	p.ctrl.Paused = false
	speaker.Unlock()

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
		
		// Pause the mixer and clear it
		speaker.Lock()
		p.ctrl.Paused = true
		p.mixer.Clear()
		speaker.Unlock()
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

	// Standard guitar tuning frequencies (in Hz)
	// E(6th) A(5th) D(4th) G(3rd) B(2nd) e(1st) - but our array is reversed
	stringFrequencies := [6]float64{329.63, 246.94, 196.00, 146.83, 110.00, 82.41} // e B G D A E (high to low as displayed)

	maxLength := tab.GetTotalLength()

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
					// Calculate frequency based on fret position
					// Each fret increases frequency by a factor of 2^(1/12)
					frequency := stringFrequencies[stringIdx] * math.Pow(2, float64(fret)/12.0)

					note := PlayableNote{
						Frequency: frequency,
						Start:     time.Duration(pos) * beatDuration,
						Duration:  beatDuration * 3 / 4, // Note length (slightly shorter than beat)
						Volume:    0.1,                  // Lower volume to prevent distortion
						String:    stringIdx,
						Position:  pos,
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
		
		// Pause and clear mixer
		speaker.Lock()
		p.ctrl.Paused = true
		p.mixer.Clear()
		speaker.Unlock()
		
		p.mu.Unlock()
		fmt.Println("Playback loop ended")
	}()

	// Use the tab's tempo
	tempo := 120
	if p.currentTab != nil && p.currentTab.Tempo > 0 {
		tempo = p.currentTab.Tempo
	}

	beatDuration := time.Minute / time.Duration(tempo*4) // 16th notes
	fmt.Printf("Starting playback: tempo=%d, beatDuration=%v\n", tempo, beatDuration)

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
		maxPos = p.currentTab.GetTotalLength()
	}

	fmt.Printf("Playback range: 0 to %d positions\n", maxPos)

	startTime := time.Now()

	for {
		select {
		case <-p.stopChan:
			fmt.Println("Playback stopped by user")
			return
		case <-ticker.C:
			p.mu.Lock()

			// Update playback time
			p.playbackTime = time.Since(startTime)

			// Update highlighted positions based on current position
			p.highlighted = nil
			notesAtPosition := 0
			for _, note := range p.notes {
				if note.Position == p.position {
					p.highlighted = append(p.highlighted, models.Position{
						String:   note.String,
						Position: note.Position,
					})
					notesAtPosition++

					// Play the note using beep
					p.playNote(note)
				}
			}

			if notesAtPosition > 0 {
				fmt.Printf("Position %d: playing %d notes\n", p.position, notesAtPosition)
			}

			p.position++

			// Check if we've reached the end
			if p.position > maxPos {
				fmt.Println("Reached end of tab")
				p.mu.Unlock()
				return
			}

			p.mu.Unlock()
		}
	}
}

func (p *Player) playNote(note PlayableNote) {
	// Create a sine wave generator for the note
	generator, err := generators.SinTone(p.sampleRate, int(note.Frequency))
	if err != nil {
		fmt.Printf("Error creating tone generator: %v\n", err)
		return
	}

	// Apply volume control
	volume := &effects.Volume{
		Streamer: generator,
		Base:     2,
		Volume:   note.Volume,
		Silent:   false,
	}

	// Create a limited duration streamer
	duration := p.sampleRate.N(note.Duration)
	limited := beep.Take(duration, volume)

	// Add to mixer (this will now be heard because the mixer is playing through speaker)
	speaker.Lock()
	p.mixer.Add(limited)
	speaker.Unlock()
}

func (p *Player) GetPlaybackInfo() (position int, totalLength int, isPlaying bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	maxPos := 0
	if p.currentTab != nil {
		maxPos = p.currentTab.GetTotalLength()
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
