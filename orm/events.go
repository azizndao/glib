// Package orm provides model event hooks for lifecycle management.
package orm

import "gorm.io/gorm"

// ModelEvents defines optional event hooks that models can implement.
// These hooks are called at various points in the model lifecycle.
//
// GORM Hook Execution Order:
//   - Create: BeforeSave -> BeforeCreate -> [Save to DB] -> AfterCreate -> AfterSave
//   - Update: BeforeSave -> BeforeUpdate -> [Save to DB] -> AfterUpdate -> AfterSave
//   - Delete: BeforeDelete -> [Delete from DB] -> AfterDelete
//   - Query: [Query DB] -> AfterFind
//
// Example implementation:
//
//	type User struct {
//	    orm.Model
//	    Name     string
//	    Email    string
//	    Password string
//	}
//
//	func (u *User) BeforeCreate(tx *gorm.DB) error {
//	    // Hash password before creating
//	    hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), 10)
//	    if err != nil {
//	        return err
//	    }
//	    u.Password = string(hashed)
//	    return nil
//	}
//
//	func (u *User) AfterCreate(tx *gorm.DB) error {
//	    // Send welcome email after user is created
//	    go sendWelcomeEmail(u.Email)
//	    return nil
//	}
//
//	func (u *User) BeforeDelete(tx *gorm.DB) error {
//	    // Prevent deletion of admin users
//	    if u.IsAdmin() {
//	        return errors.New("cannot delete admin user")
//	    }
//	    return nil
//	}
type ModelEvents interface {
	// BeforeSave is called before saving (create or update).
	BeforeSave(tx *gorm.DB) error

	// AfterSave is called after saving (create or update).
	AfterSave(tx *gorm.DB) error

	// BeforeCreate is called before creating a new record.
	BeforeCreate(tx *gorm.DB) error

	// AfterCreate is called after creating a new record.
	AfterCreate(tx *gorm.DB) error

	// BeforeUpdate is called before updating an existing record.
	BeforeUpdate(tx *gorm.DB) error

	// AfterUpdate is called after updating an existing record.
	AfterUpdate(tx *gorm.DB) error

	// BeforeDelete is called before deleting a record.
	BeforeDelete(tx *gorm.DB) error

	// AfterDelete is called after deleting a record.
	AfterDelete(tx *gorm.DB) error

	// AfterFind is called after finding a record(s).
	AfterFind(tx *gorm.DB) error
}

// Note: You don't need to implement all methods. GORM only calls hooks
// that are actually implemented on your model. Just implement the hooks
// you need:
//
//	type Post struct {
//	    orm.Model
//	    Title     string
//	    Published bool
//	}
//
//	// Only implement the hooks you need
//	func (p *Post) BeforeCreate(tx *gorm.DB) error {
//	    // Validate title
//	    if p.Title == "" {
//	        return errors.New("title is required")
//	    }
//	    return nil
//	}
//
// GORM will automatically call your BeforeCreate hook when you create
// a Post, but won't complain about missing AfterCreate, BeforeUpdate, etc.
