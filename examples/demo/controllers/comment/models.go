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
	Content string    `json:"content" validate:"required,min=1,max=1000"`
	PostID  uuid.UUID `json:"post_id" validate:"required,uuid4"`
	UserID  uuid.UUID `json:"user_id" validate:"required,uuid4"`
}

func (r CreateCommentRequest) Validate() bool {
	return true
}

type UpdateCommentRequest struct {
	Content string `json:"content" validate:"omitempty,min=1,max=1000"`
}
