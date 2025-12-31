package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID           uuid.UUID      `gorm:"type:text;primary_key" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	Username     string         `gorm:"uniqueIndex;not null" json:"username"`
	PasswordHash string         `gorm:"not null" json:"-"` // Never expose password in JSON
	FirstName    string         `gorm:"size:100" json:"first_name"`
	LastName     string         `gorm:"size:100" json:"last_name"`
	Bio          string         `gorm:"type:text" json:"bio,omitempty"`
	Active       bool           `gorm:"default:true" json:"active"`
	Posts        []Post         `gorm:"foreignKey:AuthorID" json:"posts,omitempty"`
	Comments     []Comment      `gorm:"foreignKey:UserID" json:"comments,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) FullName() string {
	return u.FirstName + " " + u.LastName
}
