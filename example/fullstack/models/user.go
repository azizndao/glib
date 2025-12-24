package models

import (
	"github.com/azizndao/glib/database/orm"
)

// User represents a user in the system
type User struct {
	orm.Model
	Name     string    `json:"name" gorm:"type:varchar(100);not null"`
	Email    string    `json:"email" gorm:"uniqueIndex;not null"`
	Password string    `json:"-" gorm:"not null"` // Never expose password in JSON
	Posts    []Post    `json:"posts,omitempty" gorm:"foreignKey:UserID"`
	Comments []Comment `json:"comments,omitempty" gorm:"foreignKey:UserID"`
}

// TableName overrides the default table name
func (User) TableName() string {
	return "users"
}
