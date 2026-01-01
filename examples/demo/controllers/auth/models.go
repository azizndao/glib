package auth

import (
	"time"

	"github.com/google/uuid"
)

// UserResponse represents the authenticated user data returned to clients
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Bio       string    `json:"bio,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RegisterRequest represents user registration data
type RegisterRequest struct {
	Email     string `json:"email" validate:"required,email"`
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Bio       string `json:"bio,omitempty"`
}

func (r RegisterRequest) Validate() bool {
	return true
}

type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (r LoginRequest) Validate() bool { return true }

type LoginResponse struct {
	User  *UserResponse `json:"user"`
	Token string        `json:"token,omitempty"` // Optional: for JWT implementation
}

func (r LoginResponse) Validate() bool { return true }

type UpdateProfileRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Bio       *string `json:"bio,omitempty"`
}

func (r UpdateProfileRequest) Validate() bool { return true }
