package orm

import "gorm.io/gorm"

// Scope is a function that modifies a GORM query.
type Scope func(*gorm.DB) *gorm.DB

// Scope factory functions - users create scopes for their specific columns

// WhereColumn returns a scope that filters by a column value.
// Example: WhereColumn("active", true), WhereColumn("status", "published")
func WhereColumn(column string, value any) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(column+" = ?", value)
	}
}

// WhereNotColumn returns a scope that filters where column doesn't match value.
func WhereNotColumn(column string, value any) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(column+" != ?", value)
	}
}

// OrderByCreatedAt orders by created_at in the specified direction.
func OrderByCreatedAt(direction string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order("created_at " + direction)
	}
}

// OrderByUpdatedAt orders by updated_at in the specified direction.
func OrderByUpdatedAt(direction string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order("updated_at " + direction)
	}
}

// OrderByColumn returns a scope that orders by the specified column and direction.
func OrderByColumn(column, direction string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Order(column + " " + direction)
	}
}

// PaginateScope returns a scope that paginates results.
func PaginateScope(page, perPage int) Scope {
	return func(db *gorm.DB) *gorm.DB {
		offset := (page - 1) * perPage
		return db.Offset(offset).Limit(perPage)
	}
}

// Search returns a scope that searches across multiple columns.
func Search(columns []string, term string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		if term == "" {
			return db
		}

		query := db
		for i, col := range columns {
			if i == 0 {
				query = query.Where(col+" LIKE ?", "%"+term+"%")
			} else {
				query = query.Or(col+" LIKE ?", "%"+term+"%")
			}
		}
		return query
	}
}

// WithRelations returns a scope that preloads the specified relations.
func WithRelations(relations ...string) Scope {
	return func(db *gorm.DB) *gorm.DB {
		for _, rel := range relations {
			db = db.Preload(rel)
		}
		return db
	}
}

// BelongsTo returns records where foreign_key matches the given ID.
// The ID can be any type (uuid.UUID, uint, string, etc.)
func BelongsTo(foreignKey string, id any) Scope {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where(foreignKey+" = ?", id)
	}
}
