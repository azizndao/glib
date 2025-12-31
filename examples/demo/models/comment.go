package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Comment struct {
	ID        uuid.UUID      `gorm:"type:text;primary_key" json:"id"`
	PostID    uuid.UUID      `gorm:"type:text;not null;index" json:"post_id"`
	UserID    uuid.UUID      `gorm:"type:text;not null;index" json:"user_id"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	Post      *Post          `gorm:"foreignKey:PostID" json:"post,omitempty"`
	User      *User          `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (Comment) TableName() string {
	return "comments"
}
