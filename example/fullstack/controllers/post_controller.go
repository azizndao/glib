package controllers

import (
	"glib/example/fullstack/middleware"
	"glib/example/fullstack/models"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/common/container"
	"github.com/azizndao/glib/common/errors"
	"github.com/azizndao/glib/database"
	"github.com/azizndao/glib/database/orm"
	"github.com/azizndao/glib/foundation"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PostController handles blog post operations.
// Implements APIResourceController interface.
type PostController struct {
	db *gorm.DB
}

// NewPostController creates a new post controller with dependencies injected.
func NewPostController(app *foundation.Application) *PostController {
	dbManager, _ := container.Resolve[*database.Manager](app.Container())
	conn, _ := dbManager.DB()

	return &PostController{
		db: conn.DB(),
	}
}

// CreatePostRequest represents the post creation payload.
type CreatePostRequest struct {
	Title   string `json:"title" validate:"required,min=5,max=200"`
	Content string `json:"content" validate:"required,min=10"`
}

// UpdatePostRequest represents the post update payload.
type UpdatePostRequest struct {
	Title     *string `json:"title" validate:"omitempty,min=5,max=200"`
	Content   *string `json:"content" validate:"omitempty,min=10"`
	Published *bool   `json:"published"`
}

// Index lists all published posts with pagination.
// GET /posts
func (ctrl *PostController) Index(c *glib.Ctx) error {
	ctx := c.Request.Context()

	// Get query parameters
	page := 1
	perPage := 10

	// Build query using ORM generics
	chain := orm.G[models.Post](ctrl.db).
		Where("published = ?", true).
		Order("created_at DESC")

	// Paginate
	paginator, err := orm.Paginate(ctx, chain, page, perPage)
	if err != nil {
		return errors.InternalServerError("Failed to fetch posts", err)
	}

	return c.JSON(paginator)
}

// Store creates a new post.
// POST /posts
func (ctrl *PostController) Store(c *glib.Ctx) error {
	var req CreatePostRequest
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	post := models.Post{
		Title:   req.Title,
		Content: req.Content,
		UserID:  userID,
	}

	if err := ctrl.db.Create(&post).Error; err != nil {
		return errors.InternalServerError("Failed to create post", err)
	}

	// Load user relation
	ctrl.db.Preload("User").First(&post, "id = ?", post.ID)

	return c.Status(201).JSON(post)
}

// Show displays a specific post with user and comments.
// GET /posts/{id}
func (ctrl *PostController) Show(c *glib.Ctx) error {
	idStr := c.PathValue("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		return errors.BadRequest("Invalid post ID", err)
	}

	// Load post with relations
	var post models.Post
	err = ctrl.db.
		Preload("User").
		Preload("Comments.User").
		Where("id = ?", postID).
		First(&post).Error

	if err != nil {
		return errors.NotFound("Post not found", err)
	}

	return c.JSON(post)
}

// Update modifies an existing post.
// PUT/PATCH /posts/{id}
func (ctrl *PostController) Update(c *glib.Ctx) error {
	idStr := c.PathValue("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		return errors.BadRequest("Invalid post ID", err)
	}

	var req UpdatePostRequest
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	// Find post (only owner can update)
	var post models.Post
	if err := ctrl.db.Where("id = ? AND user_id = ?", postID, userID).First(&post).Error; err != nil {
		return errors.NotFound("Post not found or unauthorized", err)
	}

	// Update fields
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.Published != nil {
		updates["published"] = *req.Published
	}

	if err := ctrl.db.Model(&post).Updates(updates).Error; err != nil {
		return errors.InternalServerError("Failed to update post", err)
	}

	// Reload post
	ctrl.db.Preload("User").First(&post, "id = ?", post.ID)

	return c.JSON(post)
}

// Destroy deletes a post (soft delete).
// DELETE /posts/{id}
func (ctrl *PostController) Destroy(c *glib.Ctx) error {
	idStr := c.PathValue("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		return errors.BadRequest("Invalid post ID", err)
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	// Soft delete (only owner can delete)
	result := ctrl.db.Where("id = ? AND user_id = ?", postID, userID).Delete(&models.Post{})
	if result.Error != nil {
		return errors.InternalServerError("Failed to delete post", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.NotFound("Post not found or unauthorized", nil)
	}

	return c.NoContent()
}

// AddComment adds a comment to a post.
// This is a custom action, not part of the resource controller.
// POST /posts/{id}/comments
func (ctrl *PostController) AddComment(c *glib.Ctx) error {
	idStr := c.PathValue("id")
	postID, err := uuid.Parse(idStr)
	if err != nil {
		return errors.BadRequest("Invalid post ID", err)
	}

	type CreateCommentRequest struct {
		Content string `json:"content" validate:"required,min=1,max=1000"`
	}

	var req CreateCommentRequest
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return err
	}

	// Verify post exists
	var post models.Post
	if err := ctrl.db.Where("id = ?", postID).First(&post).Error; err != nil {
		return errors.NotFound("Post not found", err)
	}

	comment := models.Comment{
		Content: req.Content,
		PostID:  postID,
		UserID:  userID,
	}

	if err := ctrl.db.Create(&comment).Error; err != nil {
		return errors.InternalServerError("Failed to create comment", err)
	}

	// Load relations
	ctrl.db.Preload("User").First(&comment, "id = ?", comment.ID)

	return c.Status(201).JSON(comment)
}
