// main.go
package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Cod-e-Codes/tuitar/internal/storage"
	"github.com/Cod-e-Codes/tuitar/internal/ui"
)

func main() {
	// Initialize storage
	storage, err := storage.NewSQLiteStorage("tabs.db")
	if err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	// Create the main application model
	m := ui.NewModel(storage)

	// Start the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v", err)
		os.Exit(1)
	}
}
