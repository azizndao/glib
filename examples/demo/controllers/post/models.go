package post

import "github.com/google/uuid"

type CreatePostRequest struct {
	Title     string    `json:"title" validate:"required,min=3,max=200"`
	Body      string    `json:"body" validate:"required,min=10"`
	Slug      string    `json:"slug" validate:"omitempty,max=100"`
	Published bool      `json:"published"`
	AuthorID  uuid.UUID `json:"author_id" validate:"required,uuid4"`
	Tags      string    `json:"tags" validate:"omitempty,max=500"`
}

type UpdatePostRequest struct {
	Title     string `json:"title" validate:"omitempty,min=3,max=200"`
	Body      string `json:"body" validate:"omitempty,min=10"`
	Slug      string `json:"slug" validate:"omitempty,max=100"`
	Published *bool  `json:"published"`
	Tags      string `json:"tags" validate:"omitempty,max=500"`
}
