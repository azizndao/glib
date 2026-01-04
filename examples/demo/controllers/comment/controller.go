package comment

import (
	"context"
	"glib/demo/models"
	"glib/demo/services"

	"github.com/google/uuid"
)

// @Controller path=/api/v1/comment tags=api
type Controller struct {
	Logger         *services.Logger // Transient provider - each controller gets fresh instance
	CommentService *services.CommentService
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) ([]models.Comment, error) {
	c.Logger.Info("Fetching all comments")
	comments, err := c.CommentService.GetComments()
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) (*models.Comment, error) {
	comment, err := c.CommentService.GetComment(id)
	if err != nil {
		return nil, err
	}
	return comment, nil
}

// @Route method=POST path=/ tags=protected
func (c *Controller) Create(ctx context.Context, req CreateCommentRequest) (*models.Comment, error) {
	comment := &models.Comment{
		Content: req.Content,
		PostID:  req.PostID,
		UserID:  req.UserID,
	}

	if err := c.CommentService.CreateComment(comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// @Route method=PUT path=/{id} tags=protected
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdateCommentRequest) (*models.Comment, error) {
	comment, err := c.CommentService.GetComment(id)
	if err != nil {
		return nil, err
	}

	if req.Content != "" {
		comment.Content = req.Content
	}

	if err := c.CommentService.UpdateComment(comment); err != nil {
		return nil, err
	}

	return comment, nil
}

// @Route method=DELETE path=/{id} tags=protected
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) error {
	if err := c.CommentService.DeleteComment(id); err != nil {
		return err
	}
	return nil
}
