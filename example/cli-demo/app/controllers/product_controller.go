package controllers

import (
	"github.com/azizndao/glib"
)

// ProductController handles HTTP requests for product resources
type ProductController struct{}

// Index lists all products
// GET /products
func (ctrl *ProductController) Index(c *glib.Ctx) error {
	// TODO: Implement
	return c.JSON(map[string]string{"message": "Index"})
}

// Show displays a specific product
// GET /products/{id}
func (ctrl *ProductController) Show(c *glib.Ctx) error {
	id := c.PathValue("id")
	// TODO: Implement
	return c.JSON(map[string]string{"id": id})
}

// Store creates a new product
// POST /products
func (ctrl *ProductController) Store(c *glib.Ctx) error {
	// TODO: Implement
	return c.Status(201).JSON(map[string]string{"message": "Created"})
}

// Update modifies an existing product
// PUT /products/{id}
func (ctrl *ProductController) Update(c *glib.Ctx) error {
	id := c.PathValue("id")
	// TODO: Implement
	return c.JSON(map[string]string{"id": id, "message": "Updated"})
}

// Destroy deletes a product
// DELETE /products/{id}
func (ctrl *ProductController) Destroy(c *glib.Ctx) error {
	_ = c.PathValue("id")
	// TODO: Implement
	return c.NoContent()
}
