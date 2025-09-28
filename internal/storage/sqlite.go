// internal/storage/sqlite.go
package storage

import (
	"database/sql"
	"encoding/json"
	"time"

	_ "modernc.org/sqlite" // SQLite driver

	"github.com/Cod-e-Codes/tuitar/internal/models"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	storage := &SQLiteStorage{db: db}
	if err := storage.migrate(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *SQLiteStorage) migrate() error {
	// Create the table if it doesn't exist
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS tabs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		artist TEXT DEFAULT '',
		content TEXT NOT NULL,
		tuning TEXT NOT NULL,
		tempo INTEGER DEFAULT 120,
		time_signature TEXT DEFAULT '4/4',
		measures INTEGER DEFAULT 4,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_tabs_name ON tabs(name);
	CREATE INDEX IF NOT EXISTS idx_tabs_updated_at ON tabs(updated_at DESC);
	`

	_, err := s.db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	// Add measures column if it doesn't exist (for existing databases)
	alterQuery := `ALTER TABLE tabs ADD COLUMN measures INTEGER DEFAULT 4;`
	_, _ = s.db.Exec(alterQuery) // Ignore error if column already exists

	return nil
}

func (s *SQLiteStorage) SaveTab(tab *models.Tab) error {
	contentJSON, _ := json.Marshal(tab.Content)
	tuningJSON, _ := json.Marshal(tab.Tuning)

	if tab.ID == 0 {
		// Insert new tab
		query := `
			INSERT INTO tabs (name, artist, content, tuning, tempo, time_signature, measures, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`
		result, err := s.db.Exec(query, tab.Name, tab.Artist, contentJSON, tuningJSON,
			tab.Tempo, tab.TimeSignature, tab.Measures, tab.CreatedAt, time.Now())
		if err != nil {
			return err
		}

		id, _ := result.LastInsertId()
		tab.ID = int(id)
	} else {
		// Update existing tab
		query := `
			UPDATE tabs SET name=?, artist=?, content=?, tuning=?, tempo=?, 
			time_signature=?, measures=?, updated_at=? WHERE id=?
		`
		_, err := s.db.Exec(query, tab.Name, tab.Artist, contentJSON, tuningJSON,
			tab.Tempo, tab.TimeSignature, tab.Measures, time.Now(), tab.ID)
		if err != nil {
			return err
		}
	}

	tab.UpdatedAt = time.Now()
	return nil
}

func (s *SQLiteStorage) LoadTab(id int) (*models.Tab, error) {
	// Use explicit column order to match our struct
	query := `SELECT id, name, artist, content, tuning, tempo, time_signature, measures, created_at, updated_at FROM tabs WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var tab models.Tab
	var contentJSON, tuningJSON string

	err := row.Scan(&tab.ID, &tab.Name, &tab.Artist, &contentJSON, &tuningJSON,
		&tab.Tempo, &tab.TimeSignature, &tab.Measures, &tab.CreatedAt, &tab.UpdatedAt)
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(contentJSON), &tab.Content)
	_ = json.Unmarshal([]byte(tuningJSON), &tab.Tuning)

	// Set default measures if not set
	if tab.Measures == 0 {
		tab.Measures = 4
	}

	return &tab, nil
}

func (s *SQLiteStorage) LoadAllTabs() ([]models.Tab, error) {
	// Use explicit column order to match our struct
	query := `SELECT id, name, artist, content, tuning, tempo, time_signature, measures, 
		created_at, updated_at FROM tabs ORDER BY updated_at DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tabs []models.Tab
	for rows.Next() {
		var tab models.Tab
		var contentJSON, tuningJSON string

		err := rows.Scan(&tab.ID, &tab.Name, &tab.Artist, &contentJSON, &tuningJSON,
			&tab.Tempo, &tab.TimeSignature, &tab.Measures, &tab.CreatedAt, &tab.UpdatedAt)
		if err != nil {
			continue
		}

		_ = json.Unmarshal([]byte(contentJSON), &tab.Content)
		_ = json.Unmarshal([]byte(tuningJSON), &tab.Tuning)

		// Set default measures if not set
		if tab.Measures == 0 {
			tab.Measures = 4
		}

		tabs = append(tabs, tab)
	}

	return tabs, nil
}

func (s *SQLiteStorage) DeleteTab(id int) error {
	query := `DELETE FROM tabs WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

func (s *SQLiteStorage) SearchTabs(query string) ([]models.Tab, error) {
	// Use explicit column order to match our struct
	sqlQuery := `
		SELECT id, name, artist, content, tuning, tempo, time_signature, measures, created_at, updated_at FROM tabs 
		WHERE name LIKE ? OR artist LIKE ? 
		ORDER BY updated_at DESC
	`

	searchTerm := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, searchTerm, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tabs []models.Tab
	for rows.Next() {
		var tab models.Tab
		var contentJSON, tuningJSON string

		err := rows.Scan(&tab.ID, &tab.Name, &tab.Artist, &contentJSON, &tuningJSON,
			&tab.Tempo, &tab.TimeSignature, &tab.Measures, &tab.CreatedAt, &tab.UpdatedAt)
		if err != nil {
			continue
		}

		_ = json.Unmarshal([]byte(contentJSON), &tab.Content)
		_ = json.Unmarshal([]byte(tuningJSON), &tab.Tuning)

		// Set default measures if not set
		if tab.Measures == 0 {
			tab.Measures = 4
		}

		tabs = append(tabs, tab)
	}

	return tabs, nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
