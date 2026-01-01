package validator

// Validable is an interface for request structs that need validation.
// If a request struct implements this interface and Validate() returns true,
// the framework will automatically validate the struct using validator tags.
// If Validate() returns false, validation is skipped.
//
// Example:
//
//	type CreatePostRequest struct {
//	    Title string `json:"title" validate:"required,min=3"`
//	    Body  string `json:"body" validate:"required"`
//	}
//
//	func (r CreatePostRequest) Validate() bool {
//	    return true // Enable validation
//	}
type Validable interface {
	Validate() bool
}
