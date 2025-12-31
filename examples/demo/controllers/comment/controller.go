package comment

import (
	"context"
	"glib/demo/models"
	"glib/demo/services"

	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

// @Controller path=/api/v1/comment tags=api
type Controller struct {
	Logger         *services.Logger // Transient provider - each controller gets fresh instance
	CommentService *services.CommentService
}

// @Route method=GET path=/
func (c *Controller) Index(ctx context.Context) glib.Result[[]models.Comment] {
	c.Logger.Info("Fetching all comments")
	comments, err := c.CommentService.GetComments()
	if err != nil {
		return glib.Fail[[]models.Comment](err)
	}
	return glib.OK(comments)
}

// @Route method=GET path=/{id}
func (c *Controller) Show(ctx context.Context, id uuid.UUID) glib.Result[*models.Comment] {
	comment, err := c.CommentService.GetComment(id)
	if err != nil {
		return glib.Fail[*models.Comment](err)
	}
	return glib.OK(comment)
}

// @Route method=POST path=/ tags=protected
func (c *Controller) Create(ctx context.Context, req CreateCommentRequest) glib.Result[*models.Comment] {
	comment := &models.Comment{
		Content: req.Content,
		PostID:  req.PostID,
		UserID:  req.UserID,
	}

	if err := c.CommentService.CreateComment(comment); err != nil {
		return glib.Fail[*models.Comment](err)
	}

	return glib.Created(comment)
}

// @Route method=PUT path=/{id} tags=protected
func (c *Controller) Update(ctx context.Context, id uuid.UUID, req UpdateCommentRequest) glib.Result[*models.Comment] {
	comment, err := c.CommentService.GetComment(id)
	if err != nil {
		return glib.Fail[*models.Comment](err)
	}

	if req.Content != "" {
		comment.Content = req.Content
	}

	if err := c.CommentService.UpdateComment(comment); err != nil {
		return glib.Fail[*models.Comment](err)
	}

	return glib.OK(comment)
}

// @Route method=DELETE path=/{id} tags=protected
func (c *Controller) Delete(ctx context.Context, id uuid.UUID) glib.Result[any] {
	if err := c.CommentService.DeleteComment(id); err != nil {
		return glib.Fail[any](err)
	}
	return glib.NoContent[any]()
}
