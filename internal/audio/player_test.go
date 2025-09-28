package audio

import (
	"testing"
	"time"

	"github.com/gopxl/beep"
)

func TestKarplusStrong(t *testing.T) {
	// Test basic Karplus-Strong synthesis
	frequency := 440.0 // A4 note
	sampleRate := beep.SampleRate(44100)
	duration := 100 * time.Millisecond

	ks := NewKarplusStrong(frequency, sampleRate, duration)

	// Test that it generates samples
	samples := make([][2]float64, 100)
	n, ok := ks.Stream(samples)

	if n == 0 {
		t.Error("Expected to generate samples, got 0")
	}

	if !ok {
		t.Error("Expected stream to be ok, got false")
	}

	// Test that samples are not all zero
	allZero := true
	for _, sample := range samples[:n] {
		if sample[0] != 0 || sample[1] != 0 {
			allZero = false
			break
		}
	}

	if allZero {
		t.Error("Expected non-zero samples, got all zeros")
	}
}

func TestKarplusStrongFrequency(t *testing.T) {
	// Test different frequencies
	frequencies := []float64{220.0, 440.0, 880.0}
	sampleRate := beep.SampleRate(44100)
	duration := 50 * time.Millisecond

	for _, freq := range frequencies {
		ks := NewKarplusStrong(freq, sampleRate, duration)
		samples := make([][2]float64, 50)
		n, ok := ks.Stream(samples)

		if n == 0 {
			t.Errorf("Expected to generate samples for frequency %f, got 0", freq)
		}

		if !ok {
			t.Errorf("Expected stream to be ok for frequency %f, got false", freq)
		}
	}
}

func TestKarplusStrongDuration(t *testing.T) {
	// Test that the synthesizer respects duration
	frequency := 440.0
	sampleRate := beep.SampleRate(44100)
	duration := 10 * time.Millisecond

	ks := NewKarplusStrong(frequency, sampleRate, duration)

	// Generate samples until stream ends
	totalSamples := 0
	for {
		samples := make([][2]float64, 100)
		n, ok := ks.Stream(samples)
		totalSamples += n

		if !ok {
			break
		}
	}

	// Should generate approximately the right number of samples
	expectedSamples := int(float64(sampleRate) * duration.Seconds())
	tolerance := expectedSamples / 10 // 10% tolerance

	if totalSamples < expectedSamples-tolerance || totalSamples > expectedSamples+tolerance {
		t.Errorf("Expected approximately %d samples, got %d", expectedSamples, totalSamples)
	}
}
