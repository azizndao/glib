package models

import (
	"github.com/azizndao/glib/orm"
)

// Product represents a product record
type Product struct {
	orm.Model
}


// TableName overrides the default table name
func (Product) TableName() string {
	return "products"
}

