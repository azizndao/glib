package comment

import (
	"time"

	"github.com/google/uuid"
)

type Comment struct {
	ID        uuid.UUID `json:"id"`
	PostID    uuid.UUID `json:"post_id"`
	UserID    uuid.UUID `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateCommentRequest struct {
	Content string    `json:"content" binding:"required"`
	PostID  uuid.UUID `json:"post_id" binding:"required"`
	UserID  uuid.UUID `json:"user_id" binding:"required"`
}

type UpdateCommentRequest struct {
	Content string `json:"content"`
}
