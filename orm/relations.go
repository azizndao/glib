// Package orm provides relationship helpers for working with GORM associations.
//
// Relationships are defined using GORM struct tags and loaded using Preload.
// This file provides helper functions to make working with relationships easier.
//
// # Relationship Types
//
// **Has One (1:1)**
//
//	type User struct {
//	    orm.Model
//	    Name    string
//	    Profile *Profile `gorm:"foreignKey:UserID"`
//	}
//
//	type Profile struct {
//	    orm.Model
//	    UserID uint
//	    Bio    string
//	}
//
//	// Load with relationship
//	user, err := orm.G[User](db).Preload("Profile").Where("id = ?", 1).First(ctx)
//
// **Has Many (1:N)**
//
//	type User struct {
//	    orm.Model
//	    Name  string
//	    Posts []Post `gorm:"foreignKey:UserID"`
//	}
//
//	type Post struct {
//	    orm.Model
//	    UserID uint
//	    Title  string
//	}
//
//	// Load with relationships
//	user, err := orm.G[User](db).Preload("Posts").Where("id = ?", 1).First(ctx)
//
// **Belongs To (Inverse)**
//
//	type Post struct {
//	    orm.Model
//	    UserID uint
//	    Title  string
//	    User   *User `gorm:"foreignKey:UserID"`
//	}
//
//	// Load post with user
//	post, err := orm.G[Post](db).Preload("User").Where("id = ?", 1).First(ctx)
//
// **Many to Many (N:M)**
//
//	type User struct {
//	    orm.Model
//	    Name  string
//	    Roles []Role `gorm:"many2many:user_roles;"`
//	}
//
//	type Role struct {
//	    orm.Model
//	    Name  string
//	}
//
//	// Load with roles
//	user, err := orm.G[User](db).Preload("Roles").Where("id = ?", 1).First(ctx)
//
// **Polymorphic Relations**
//
//	type Comment struct {
//	    orm.Model
//	    CommentableID   uint
//	    CommentableType string
//	    Body            string
//	}
//
//	type Post struct {
//	    orm.Model
//	    Title    string
//	    Comments []Comment `gorm:"polymorphic:Commentable;"`
//	}
//
//	// Load with comments
//	post, err := orm.G[Post](db).Preload("Comments").Where("id = ?", 1).First(ctx)
//
// # Eager Loading (Prevent N+1 Queries)
//
// **Basic Preload**
//
//	users, err := orm.G[User](db).Preload("Posts").Find(ctx)
//
// **Nested Preload**
//
//	users, err := orm.G[User](db).
//	    Preload("Posts.Comments").
//	    Preload("Profile").
//	    Find(ctx)
//
// **Conditional Preload**
//
//	users, err := orm.G[User](db).
//	    Preload("Posts", "published = ?", true).
//	    Find(ctx)
//
// **Custom Preload Query**
//
//	users, err := orm.G[User](db).
//	    Preload("Posts", func(db *gorm.DB) *gorm.DB {
//	        return db.Where("published = ?", true).
//	            Order("created_at DESC").
//	            Limit(5)
//	    }).
//	    Find(ctx)
//
// # Association Helpers
//
// The Association helpers make it easy to work with related models:
//
//	// Append related models
//	orm.Association(db, &user, "Roles").Append(&admin, &editor)
//
//	// Replace all related models
//	orm.Association(db, &user, "Roles").Replace(&admin)
//
//	// Delete specific related models
//	orm.Association(db, &user, "Roles").Delete(&editor)
//
//	// Clear all related models
//	orm.Association(db, &user, "Roles").Clear()
//
//	// Count related models
//	count := orm.Association(db, &user, "Roles").Count()
//
//	// Find related models
//	var roles []Role
//	orm.Association(db, &user, "Roles").Find(&roles)
package orm

import (
	"context"

	"gorm.io/gorm"
)

// Association returns a GORM association helper for the given model and relationship.
//
// This is a convenience wrapper around db.Model(model).Association(relation).
//
// Example:
//
//	// Append roles to user
//	admin := &Role{Name: "admin"}
//	orm.Association(db, &user, "Roles").Append(admin)
//
//	// Count user's roles
//	count := orm.Association(db, &user, "Roles").Count()
//
//	// Replace all roles
//	orm.Association(db, &user, "Roles").Replace(&role1, &role2)
//
//	// Clear all roles
//	orm.Association(db, &user, "Roles").Clear()
func Association(db *gorm.DB, model any, relation string) *gorm.Association {
	return db.Model(model).Association(relation)
}

// PreloadRelations is a helper to preload multiple relations at once.
//
// Example:
//
//	users, err := orm.G[User](orm.PreloadRelations(db, "Posts", "Profile", "Roles")).Find(ctx)
//
//	// With conditions
//	users, err := orm.G[User](orm.PreloadRelations(db,
//	    "Posts", func(db *gorm.DB) *gorm.DB {
//	        return db.Where("published = ?", true)
//	    },
//	    "Profile",
//	    "Roles",
//	)).Find(ctx)
func PreloadRelations(db *gorm.DB, relations ...any) *gorm.DB {
	for i := 0; i < len(relations); i++ {
		if relation, ok := relations[i].(string); ok {
			// Check if next item is a function or args
			if i+1 < len(relations) {
				switch next := relations[i+1].(type) {
				case func(*gorm.DB) *gorm.DB:
					db = db.Preload(relation, next)
					i++ // Skip the function
					continue
				case string:
					// Next is another relation name
					db = db.Preload(relation)
					continue
				default:
					// Might be condition args
					// Collect args until we hit next string or function
					var args []any
					for j := i + 1; j < len(relations); j++ {
						if _, ok := relations[j].(string); ok {
							break
						}
						if _, ok := relations[j].(func(*gorm.DB) *gorm.DB); ok {
							break
						}
						args = append(args, relations[j])
						i++
					}
					if len(args) > 0 {
						db = db.Preload(relation, args...)
					} else {
						db = db.Preload(relation)
					}
				}
			} else {
				db = db.Preload(relation)
			}
		}
	}
	return db
}

// HasOneDoc is a documentation helper showing how to define a HasOne relationship.
//
// A HasOne relationship indicates that a model has at most one of another model.
//
// Example:
//
//	type User struct {
//	    orm.Model
//	    Name    string
//	    Profile *Profile `gorm:"foreignKey:UserID"`
//	}
//
//	type Profile struct {
//	    orm.Model
//	    UserID uint    // Foreign key pointing to User.ID
//	    Bio    string
//	}
//
//	// Query with relationship
//	user, err := orm.G[User](db).Preload("Profile").First(ctx)
//	fmt.Println(user.Profile.Bio)
func HasOneDoc() {
	// This is a documentation-only function
	panic("HasOneDoc is a documentation helper. Use GORM struct tags to define relationships.")
}

// HasManyDoc is a documentation helper showing how to define a HasMany relationship.
//
// A HasMany relationship indicates that a model has zero or more of another model.
//
// Example:
//
//	type User struct {
//	    orm.Model
//	    Name  string
//	    Posts []Post `gorm:"foreignKey:UserID"`
//	}
//
//	type Post struct {
//	    orm.Model
//	    UserID uint    // Foreign key pointing to User.ID
//	    Title  string
//	}
//
//	// Query with relationship
//	user, err := orm.G[User](db).Preload("Posts").First(ctx)
//	for _, post := range user.Posts {
//	    fmt.Println(post.Title)
//	}
//
//	// Conditional preload
//	user, err := orm.G[User](db).
//	    Preload("Posts", "published = ?", true).
//	    First(ctx)
func HasManyDoc() {
	// This is a documentation-only function
	panic("HasManyDoc is a documentation helper. Use GORM struct tags to define relationships.")
}

// BelongsToDoc is a documentation helper showing how to define a BelongsTo relationship.
//
// A BelongsTo relationship is the inverse of HasOne or HasMany.
//
// Example:
//
//	type Post struct {
//	    orm.Model
//	    UserID uint
//	    Title  string
//	    User   *User `gorm:"foreignKey:UserID"`
//	}
//
//	type User struct {
//	    orm.Model
//	    Name string
//	}
//
//	// Query with relationship
//	post, err := orm.G[Post](db).Preload("User").First(ctx)
//	fmt.Println(post.User.Name)
func BelongsToDoc() {
	// This is a documentation-only function
	panic("BelongsToDoc is a documentation helper. Use GORM struct tags to define relationships.")
}

// ManyToManyDoc is a documentation helper showing how to define a ManyToMany relationship.
//
// A ManyToMany relationship indicates that a model can have many of another model,
// and vice versa, through a pivot table.
//
// Example:
//
//	type User struct {
//	    orm.Model
//	    Name  string
//	    Roles []Role `gorm:"many2many:user_roles;"`
//	}
//
//	type Role struct {
//	    orm.Model
//	    Name string
//	}
//
//	// The pivot table "user_roles" will have columns: user_id, role_id
//
//	// Query with relationship
//	user, err := orm.G[User](db).Preload("Roles").First(ctx)
//	for _, role := range user.Roles {
//	    fmt.Println(role.Name)
//	}
//
//	// Attach role to user
//	orm.Association(db, &user, "Roles").Append(&admin)
//
//	// Detach role from user
//	orm.Association(db, &user, "Roles").Delete(&admin)
//
//	// Replace all roles
//	orm.Association(db, &user, "Roles").Replace(&role1, &role2)
//
//	// Clear all roles
//	orm.Association(db, &user, "Roles").Clear()
func ManyToManyDoc() {
	// This is a documentation-only function
	panic("ManyToManyDoc is a documentation helper. Use GORM struct tags to define relationships.")
}

// LoadWith is a helper that combines model querying with relationship preloading.
//
// Example:
//
//	// Load user with all relationships
//	user, err := orm.LoadWith[User](ctx, db, 1, "Posts", "Profile", "Roles")
//
//	// Load user with conditional preloading
//	user, err := orm.LoadWith[User](ctx, db, 1,
//	    "Posts", func(db *gorm.DB) *gorm.DB {
//	        return db.Where("published = ?", true).Order("created_at DESC")
//	    },
//	    "Profile",
//	)
func LoadWith[T any](ctx context.Context, db *gorm.DB, id any, relations ...any) (T, error) {
	query := db
	if len(relations) > 0 {
		query = PreloadRelations(db, relations...)
	}
	return G[T](query).Where("id = ?", id).First(ctx)
}

// LoadMany is like LoadWith but loads multiple records.
//
// Example:
//
//	users, err := orm.LoadMany[User](ctx, db,
//	    func(db *gorm.DB) *gorm.DB {
//	        return db.Where("active = ?", true)
//	    },
//	    "Posts", "Profile", "Roles",
//	)
func LoadMany[T any](ctx context.Context, db *gorm.DB, queryFn func(*gorm.DB) *gorm.DB, relations ...any) ([]T, error) {
	query := db
	if queryFn != nil {
		query = queryFn(db)
	}
	if len(relations) > 0 {
		query = PreloadRelations(query, relations...)
	}
	return G[T](query).Find(ctx)
}
