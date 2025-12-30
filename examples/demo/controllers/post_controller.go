package controllers

import "context"

// @Controller
type PostController struct{}

// @Route GET /hello
func Hello(ctx context.Context) (string, error) {
	return "Hello WOrld", nil
}
