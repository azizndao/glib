package controllers

import "context"

// PostController is a controller for post
// @Controller
type PostController struct{}

// Hello returns hello world
// @Route GET /hello
func (c *PostController) Hello(ctx context.Context) (string, error) {
	return "Hello WOrld", nil
}
