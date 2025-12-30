package admin

import (
	"time"
	
	"github.com/google/uuid"
)

type Admin struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateAdminRequest struct {
	// TODO: add fields
}

type UpdateAdminRequest struct {
	// TODO: add fields
}
