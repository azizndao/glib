package auth

import (
	"context"
)

// Nested test models
type Address struct {
	Street     string `json:"street" validate:"required,min=5"`
	City       string `json:"city" validate:"required"`
	PostalCode string `json:"postal_code" validate:"required,len=5"`
}

type UserProfile struct {
	Name    string   `json:"name" validate:"required,min=3,max=50"`
	Age     int      `json:"age" validate:"required,gte=18"`
	Address *Address `json:"address" validate:"required"`
}

type CreateProfileRequest struct {
	User UserProfile `json:"user" validate:"required"`
}

func (r CreateProfileRequest) Validate() bool { return true }

// @Route method=POST path=/test-nested
func (c *Controller) TestNested(ctx context.Context, req CreateProfileRequest) (*UserResponse, error) {
	// Just return success for testing validation
	return nil, nil
}
