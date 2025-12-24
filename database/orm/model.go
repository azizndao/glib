// Package orm provides the ORM layer with Active Record pattern support.
package orm

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Model is the base model struct that provides common fields for all models.
// Embed this in your models to get automatic UUID primary key, timestamps, and soft delete support.
type Model struct {
	ID        uuid.UUID      `gorm:"type:char(36);primaryKey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// BeforeCreate hook to generate UUID if not set (for databases that don't support gen_random_uuid())
func (m *Model) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// IsNew returns true if the model hasn't been saved to the database yet.
func (m *Model) IsNew() bool {
	return m.ID == uuid.Nil
}

// IsDeleted returns true if the model is soft deleted.
func (m *Model) IsDeleted() bool {
	return m.DeletedAt.Valid
}

// ModelInterface defines the interface that all models should implement.
type ModelInterface interface {
	IsNew() bool
}

// Timestamps provides timestamp fields without soft delete.
type Timestamps struct {
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SoftDeletes provides soft delete functionality.
type SoftDeletes struct {
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

// IsDeleted returns true if the model is soft deleted.
func (s *SoftDeletes) IsDeleted() bool {
	return s.DeletedAt.Valid
}
