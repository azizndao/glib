package models

import (
	"github.com/azizndao/glib/database/orm"
	"github.com/google/uuid"
)

// Post represents a blog post
type Post struct {
	orm.Model
	Title     string    `json:"title" gorm:"type:varchar(200);not null"`
	Content   string    `json:"content" gorm:"type:text;not null"`
	Published bool      `json:"published" gorm:"default:false"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`
	User      *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Comments  []Comment `json:"comments,omitempty" gorm:"foreignKey:PostID"`
}

// TableName overrides the default table name
func (Post) TableName() string {
	return "posts"
}
