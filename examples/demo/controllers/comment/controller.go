package comment

import (
	"context"

	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

// @Controller path=/api/v1/comment tags=api
type Controller struct {
	// Add dependencies here (auto-injected)
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) glib.Result[[]Comment] {
	// TODO: implement
	return glib.OK([]Comment{})
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*Comment] {
	// TODO: implement
	return glib.OK(&Comment{ID: id})
}

// @Route method=POST path=/ tags=protected
func (c *Controller) Create(ctx context.Context, req CreateCommentRequest) glib.Result[*Comment] {
	// TODO: implement
	comment := &Comment{ID: uuid.New()}
	return glib.Created(comment)
}

// @Route method=PUT path=/{id} tags=protected
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdateCommentRequest) glib.Result[*Comment] {
	// TODO: implement
	return glib.OK(&Comment{ID: id})
}

// @Route method=DELETE path=/{id} tags=protected
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
	// TODO: implement
	return glib.NoContent[any]()
}
