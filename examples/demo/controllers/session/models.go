package session

import (
	"time"
	
	"github.com/google/uuid"
)

type Session struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateSessionRequest struct {
	// TODO: add fields
}

type UpdateSessionRequest struct {
	// TODO: add fields
}
