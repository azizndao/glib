package typeutil

import "github.com/azizndao/glib/router"

// ValidateBody is a generic helper to parse and validate the request body
func ValidateBody[T any](c *router.Ctx) (*T, error) {
	var out T
	if err := c.ValidateBody(&out); err != nil {
		return nil, err
	}
	return &out, nil
}
