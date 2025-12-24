package models

import (
	"github.com/azizndao/glib/database/orm"
	"github.com/google/uuid"
)

// Comment represents a comment on a post
type Comment struct {
	orm.Model
	Content string    `json:"content" gorm:"type:text;not null"`
	PostID  uuid.UUID `json:"post_id" gorm:"type:char(36);not null;index"`
	UserID  uuid.UUID `json:"user_id" gorm:"type:char(36);not null;index"`
	Post    *Post     `json:"post,omitempty" gorm:"foreignKey:PostID"`
	User    *User     `json:"user,omitempty" gorm:"foreignKey:UserID"`
}

// TableName overrides the default table name
func (Comment) TableName() string {
	return "comments"
}
