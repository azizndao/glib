package post

type CreatePostRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

type UpdatePostRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}
