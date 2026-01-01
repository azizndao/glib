package post

import (
	"context"
	"encoding/json"
	"fmt"
	"glib/demo/models"
	"glib/demo/services"
	"net/http"
	"time"

	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

// @Controller path=/api/v1/post tags=api
type Controller struct {
	UserSerivce *services.UserSerivce
	PostSerivce *services.PostSerivce
	Logger      *services.Logger // Transient provider
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context, params PostPaginationParams) glib.Result[[]models.Post] {
	c.Logger.Info("Fetching all posts")
	posts, err := c.PostSerivce.GetPosts()
	if err != nil {
		return glib.Fail[[]models.Post](err)
	}
	return glib.OK(posts)
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*models.Post] {
	post, err := c.PostSerivce.GetPost(id)
	if err != nil {
		return glib.Fail[*models.Post](err)
	}
	return glib.OK(post)
}

// @Route method=POST path=/ tags=protected
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) glib.Result[*models.Post] {
	post := &models.Post{
		Title:     req.Title,
		Body:      req.Body,
		Slug:      req.Slug,
		Published: req.Published,
		AuthorID:  req.AuthorID,
		Tags:      req.Tags,
	}

	if err := c.PostSerivce.CreatePost(post); err != nil {
		return glib.Fail[*models.Post](err)
	}

	return glib.Created(post)
}

// @Route method=PUT path=/{id} tags=protected
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdatePostRequest) glib.Result[*models.Post] {
	post, err := c.PostSerivce.GetPost(id)
	if err != nil {
		return glib.Fail[*models.Post](err)
	}

	if req.Title != "" {
		post.Title = req.Title
	}
	if req.Body != "" {
		post.Body = req.Body
	}
	if req.Slug != "" {
		post.Slug = req.Slug
	}
	if req.Published != nil {
		post.Published = *req.Published
	}
	if req.Tags != "" {
		post.Tags = req.Tags
	}

	if err := c.PostSerivce.UpdatePost(post); err != nil {
		return glib.Fail[*models.Post](err)
	}

	return glib.OK(post)
}

// @Route method=DELETE path=/{id} tags=protected
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
	if err := c.PostSerivce.DeletePost(id); err != nil {
		return glib.Fail[any](err)
	}
	return glib.NoContent[any]()
}

// @Route method=GET path=/export
// Export demonstrates Pattern 11 - raw HTTP handler for custom response handling
// This could be used for file downloads, streaming, SSE, etc.
func (c *Controller) Export(w http.ResponseWriter, r *http.Request) {
	// Example: Export posts as CSV
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=posts.csv")
	w.WriteHeader(http.StatusOK)

	// Write CSV header
	_, _ = fmt.Fprintln(w, "id,title,slug,created_at")

	// Fetch real posts from database
	posts, err := c.PostSerivce.GetPosts()
	if err != nil {
		_, _ = fmt.Fprintf(w, "# Error: %v\n", err)
		return
	}

	// Write posts data
	for _, post := range posts {
		_, _ = fmt.Fprintf(w, "%s,%q,%s,%s\n",
			post.ID,
			post.Title,
			post.Slug,
			post.CreatedAt.Format(time.RFC3339),
		)
	}
}

// @Route method=GET path=/stream
// Stream demonstrates Server-Sent Events (SSE) using raw handler
func (c *Controller) Stream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Fetch posts
	posts, err := c.PostSerivce.GetPosts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Stream posts one by one
	for _, post := range posts {
		select {
		case <-r.Context().Done():
			return
		default:
			data := map[string]any{
				"id":     post.ID,
				"title":  post.Title,
				"slug":   post.Slug,
				"author": post.AuthorID,
				"time":   time.Now().Format(time.RFC3339),
			}
			jsonData, _ := json.Marshal(data)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// @Route method=GET path=/health with=none
// Health is a simple health check endpoint with no middleware
func (c *Controller) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}
