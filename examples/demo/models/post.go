package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Post struct {
	ID        uuid.UUID      `gorm:"type:text;primary_key" json:"id"`
	Title     string         `gorm:"size:200;not null" json:"title"`
	Body      string         `gorm:"type:text;not null" json:"body"`
	Slug      string         `gorm:"uniqueIndex;size:200" json:"slug"`
	Published bool           `gorm:"default:false" json:"published"`
	AuthorID  uuid.UUID      `gorm:"type:text;not null;index" json:"author_id"`
	Author    *User          `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Comments  []Comment      `gorm:"foreignKey:PostID" json:"comments,omitempty"`
	Tags      string         `gorm:"type:text" json:"tags,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Post) TableName() string {
	return "posts"
}
