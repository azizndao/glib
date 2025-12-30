package comment

import (
	"context"
	
	"github.com/google/uuid"
)

// @Controller /api/v1/comment
type Controller struct {
	// Add dependencies here (auto-injected)
}

// @Route GET /
func (c *Controller) Index(ctx context.Context) ([]Comment, error) {
	// TODO: implement
	return nil, nil
}

// @Route GET /{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*Comment, error) {
	// TODO: implement
	return nil, nil
}

// @Route POST /
func (c *Controller) Create(ctx context.Context, req CreateCommentRequest) (*Comment, error) {
	// TODO: implement
	return nil, nil
}

// @Route PUT /{id}
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdateCommentRequest) (*Comment, error) {
	// TODO: implement
	return nil, nil
}

// @Route DELETE /{id}
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) error {
	// TODO: implement
	return nil
}
