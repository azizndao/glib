package post

import (
	"time"
	
	"github.com/google/uuid"
)

type Post struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreatePostRequest struct {
	// TODO: add fields
}

type UpdatePostRequest struct {
	// TODO: add fields
}
