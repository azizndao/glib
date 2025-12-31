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
	Email     string `json:"email" binding:"required,email"`
	Username  string `json:"username" binding:"required,min=3,max=50"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Bio       string `json:"bio,omitempty"`
}

// LoginRequest represents user login credentials
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse includes user data and authentication token
type LoginResponse struct {
	User  *UserResponse `json:"user"`
	Token string        `json:"token,omitempty"` // Optional: for JWT implementation
}

// UpdateProfileRequest represents user profile update data
type UpdateProfileRequest struct {
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Bio       *string `json:"bio,omitempty"`
}
