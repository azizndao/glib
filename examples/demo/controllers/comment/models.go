package comment

import (
	"time"
	
	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateCommentRequest struct {
	// TODO: add fields
}

type UpdateCommentRequest struct {
	// TODO: add fields
}
