// internal/storage/storage.go
package storage

import "github.com/Cod-e-Codes/tuitar/internal/models"

type Storage interface {
	SaveTab(tab *models.Tab) error
	LoadTab(id int) (*models.Tab, error)
	LoadAllTabs() ([]models.Tab, error)
	DeleteTab(id int) error
	SearchTabs(query string) ([]models.Tab, error)
}
