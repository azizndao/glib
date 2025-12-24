// Package orm provides the ORM layer with Active Record pattern support.
package orm

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// G creates a new type-safe query using GORM's generics API.
// This is a thin wrapper around gorm.G[T] for convenience.
//
// Example:
//
//	users, err := orm.G[User](db).Where("age > ?", 18).Find(ctx)
//	user, err := orm.G[User](db).Where("id = ?", 1).First(ctx)
//
// For full GORM generics API support, use gorm.G[T] directly:
//
//	users, err := gorm.G[User](db).Where("age > ?", 18).Find(ctx)
func G[T any](db *gorm.DB, opts ...clause.Expression) gorm.Interface[T] {
	return gorm.G[T](db, opts...)
}

// Paginator holds pagination information.
type Paginator[T any] struct {
	Data        []T   `json:"data"`
	Total       int64 `json:"total"`
	PerPage     int   `json:"per_page"`
	CurrentPage int   `json:"current_page"`
	LastPage    int   `json:"last_page"`
	From        int   `json:"from"`
	To          int   `json:"to"`
}

// HasMorePages checks if there are more pages.
func (p *Paginator[T]) HasMorePages() bool {
	return p.CurrentPage < p.LastPage
}

// IsEmpty checks if the paginator has no data.
func (p *Paginator[T]) IsEmpty() bool {
	return len(p.Data) == 0
}

// IsNotEmpty checks if the paginator has data.
func (p *Paginator[T]) IsNotEmpty() bool {
	return len(p.Data) > 0
}

// Paginate is a helper function for pagination.
//
// Example:
//
//	builder := orm.G[User](db).Where("active = ?", true)
//	paginator, err := orm.Paginate(ctx, builder, 1, 15)
func Paginate[T any](ctx context.Context, chain gorm.ChainInterface[T], page, perPage int) (*Paginator[T], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}

	// Get total count
	total, err := chain.Count(ctx, "*")
	if err != nil {
		return nil, err
	}

	// Calculate pagination
	offset := (page - 1) * perPage
	lastPage := int((total + int64(perPage) - 1) / int64(perPage))

	// Get data
	data, err := chain.Offset(offset).Limit(perPage).Find(ctx)
	if err != nil {
		return nil, err
	}

	return &Paginator[T]{
		Data:        data,
		Total:       total,
		PerPage:     perPage,
		CurrentPage: page,
		LastPage:    lastPage,
		From:        offset + 1,
		To:          offset + len(data),
	}, nil
}

// Chunk processes records in batches with a callback.
//
// Example:
//
//	builder := orm.G[User](db).Where("active = ?", true)
//	err := orm.Chunk(ctx, builder, 100, func(users []User) error {
//	    // Process batch
//	    return nil
//	})
func Chunk[T any](ctx context.Context, chain gorm.ChainInterface[T], batchSize int, fn func([]T) error) error {
	var offset int
	for {
		batch, err := chain.Offset(offset).Limit(batchSize).Find(ctx)
		if err != nil {
			return err
		}

		if len(batch) == 0 {
			break
		}

		if err := fn(batch); err != nil {
			return err
		}

		if len(batch) < batchSize {
			break
		}

		offset += batchSize
	}
	return nil
}

// Exists checks if any records match the query.
func Exists[T any](ctx context.Context, chain gorm.ChainInterface[T]) (bool, error) {
	count, err := chain.Count(ctx, "*")
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// DoesntExist checks if no records match the query.
func DoesntExist[T any](ctx context.Context, chain gorm.ChainInterface[T]) (bool, error) {
	exists, err := Exists(ctx, chain)
	return !exists, err
}
