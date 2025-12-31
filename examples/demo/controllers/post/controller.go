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
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) glib.Result[[]models.Post] {
	return glib.OK(
		c.PostSerivce.GetPosts(),
	)
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id int) glib.Result[*models.Post] {
	return glib.OK(c.PostSerivce.GetPost(id))
}

// @Route method=POST path=/ tags=protected
func (c *Controller) Create(ctx context.Context, req CreatePostRequest) glib.Result[*models.Post] {
	return glib.Created(&models.Post{
		ID:        c.PostSerivce.GetPosts()[len(c.PostSerivce.GetPosts())-1].ID + 1,
		Title:     req.Title,
		Body:      req.Body,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})
}

// @Route method=PUT path=/{id} tags=protected
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdatePostRequest) glib.Result[*models.Post] {
	// TODO: implement
	return glib.NotFound[*models.Post]("post not found")
}

// @Route method=DELETE path=/{id} tags=protected
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
	// TODO: implement
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
	fmt.Fprintln(w, "id,title,created_at")

	// Write sample data (in real app, fetch from DB)
	fmt.Fprintf(w, "%s,Sample Post,%s\n", uuid.New(), time.Now().Format(time.RFC3339))
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

	// Send 3 events then close
	for i := range 3 {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(1 * time.Second):
			data := map[string]any{
				"id":      uuid.New(),
				"message": fmt.Sprintf("Event %d", i+1),
				"time":    time.Now().Format(time.RFC3339),
			}
			jsonData, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
		}
	}
}
