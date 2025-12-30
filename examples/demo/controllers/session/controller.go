package session

import (
	"context"

	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

// @Controller path=/api/v1/session tags=api
type Controller struct {
	// Add dependencies here (auto-injected)
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) glib.Result[[]Session] {
	// TODO: implement
	return glib.OK([]Session{})
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id int) glib.Result[*Session] {
	// TODO: implement
	return glib.OK(&Session{ID: uuid.New()})
}

// @Route method=POST path=/
func (c *Controller) Create(ctx context.Context, req CreateSessionRequest) glib.Result[*Session] {
	// TODO: implement
	session := &Session{ID: uuid.New()}
	return glib.Created(session)
}

// @Route method=PUT path=/{id}
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdateSessionRequest) glib.Result[*Session] {
	// TODO: implement
	return glib.OK(&Session{})
}

// @Route method=DELETE path=/{id}
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
	// TODO: implement
	return glib.NoContent[any]()
}
