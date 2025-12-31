package post

import "github.com/google/uuid"

type CreatePostRequest struct {
	Title     string    `json:"title" binding:"required"`
	Body      string    `json:"body" binding:"required"`
	Slug      string    `json:"slug"`
	Published bool      `json:"published"`
	AuthorID  uuid.UUID `json:"author_id" binding:"required"`
	Tags      string    `json:"tags"`
}

type UpdatePostRequest struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	Slug      string `json:"slug"`
	Published *bool  `json:"published"`
	Tags      string `json:"tags"`
}

type SearchPostsRequest struct {
	Page   int      `query:"page"`
	Limit  int      `query:"limit"`
	Search *string  `query:"q"`
	Tags   []string `query:"tag"`
	Auth   string   `header:"Authorization"`
}
