package auth

import (
	"context"

	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

// @Controller path=/api/v1/auth tags=public
type Controller struct {
	// Add dependencies here (auto-injected)
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) glib.Result[[]Auth] {
	// TODO: implement
	return glib.OK([]Auth{})
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*Auth] {
	// TODO: implement
	return glib.OK(&Auth{ID: id})
}

// @Route method=POST path=/
func (c *Controller) Create(ctx context.Context, req CreateAuthRequest) glib.Result[*Auth] {
	// TODO: implement
	auth := &Auth{ID: uuid.New()}
	return glib.Created(auth)
}

// @Route method=PUT path=/{id}
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdateAuthRequest) glib.Result[*Auth] {
	// TODO: implement
	return glib.OK(&Auth{ID: id})
}

// @Route method=DELETE path=/{id}
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
	// TODO: implement
	return glib.NoContent[any]()
}
