package services

import (
	"context"
	"glib/demo/models"
	"uuid"

	"gorm.io/gorm"
)

type CommentService struct {
	db *gorm.DB
}

// @Provider singleton
func NewCommentService(db *gorm.DB) *CommentService {
	return &CommentService{db: db}
}

func (s *CommentService) comments() gorm.Interface[models.Comment] {
	return gorm.G[models.Comment](s.db)
}

func (s *CommentService) GetComment(ctx context.Context, id uuid.UUID) (models.Comment, error) {
	return s.comments().Preload("User", nil).Preload("Post", nil).Where("id", id).First(ctx)
}

func (s *CommentService) GetComments(ctx context.Context) ([]models.Comment, error) {
	return s.comments().Preload("User", nil).Preload("Post", nil).Order("created_at DESC").Find(ctx)
}

func (s *CommentService) GetCommentsByPost(postID uuid.UUID) ([]models.Comment, error) {
	var comments []models.Comment
	err := s.db.Preload("User").Where("post_id = ?", postID).Order("created_at DESC").Find(&comments).Error
	return comments, err
}

func (s *CommentService) CreateComment(ctx context.Context, comment *models.Comment) error {
	return s.comments().Create(ctx, comment)
}

func (s *CommentService) UpdateComment(ctx context.Context, comment models.Comment) error {
	_, err := s.comments().Where("id = ?", comment.ID).Updates(ctx, comment)
	return err
}

func (s *CommentService) DeleteComment(id uuid.UUID) error {
	return s.db.Delete(&models.Comment{}, "id = ?", id).Error
}
