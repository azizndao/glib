package auth

import (
	"context"
	"glib/demo/services"

	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

// @Controller path=/api/v1/auth tags=public
type Controller struct {
	UserSerivce *services.UserSerivce
}

// @Route method=GET path=/{id}
func (c *Controller) GetSession(ctx context.Context, id uuid.UUID) glib.Result[*Auth] {
	return glib.OK(&Auth{ID: id})
}

// @Route method=POST path=/register
func (c *Controller) Register(ctx context.Context, req CreateAuthRequest) glib.Result[*Auth] {
	auth := &Auth{ID: uuid.New()}
	return glib.Created(auth)
}

// @Route method=PUT path=/me
func (c *Controller) Update(ctx context.Context, req UpdateAuthRequest) glib.Result[*Auth] {
	return glib.NotFound[*Auth]("auth not found")
}

// @Route method=DELETE path=/logout
func (c *Controller) Delete(ctx context.Context) glib.Result[any] {
	return glib.NoContent[any]()
}
