package auth

import (
	"time"
	
	"github.com/google/uuid"
)

type Auth struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateAuthRequest struct {
	// TODO: add fields
}

type UpdateAuthRequest struct {
	// TODO: add fields
}
