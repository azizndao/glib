package admin

import (
	"context"

	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

// @Controller path=/api/v1/admin tags=admin,protected
type Controller struct {
	// Add dependencies here (auto-injected)
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) glib.Result[[]Admin] {
	// TODO: implement
	return glib.OK([]Admin{})
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*Admin] {
	// TODO: implement
	return glib.OK(&Admin{ID: id})
}

// @Route method=POST path=/
func (c *Controller) Create(ctx context.Context, req CreateAdminRequest) glib.Result[*Admin] {
	// TODO: implement
	admin := &Admin{ID: uuid.New()}
	return glib.Created(admin)
}

// @Route method=PUT path=/{id}
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdateAdminRequest) glib.Result[*Admin] {
	// TODO: implement
	return glib.OK(&Admin{ID: id})
}

// @Route method=DELETE path=/{id}
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
	// TODO: implement
	return glib.NoContent[any]()
}
