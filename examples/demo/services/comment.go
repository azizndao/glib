package services

import (
	"glib/demo/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CommentService struct {
	db *gorm.DB
}

// @Provider singleton
func NewCommentService(db *gorm.DB) *CommentService {
	return &CommentService{db: db}
}

func (s *CommentService) GetComment(id uuid.UUID) (*models.Comment, error) {
	var comment models.Comment
	err := s.db.Preload("User").Preload("Post").First(&comment, "id = ?", id).Error
	return &comment, err
}

func (s *CommentService) GetComments() ([]models.Comment, error) {
	var comments []models.Comment
	err := s.db.Preload("User").Preload("Post").Order("created_at DESC").Find(&comments).Error
	return comments, err
}

func (s *CommentService) GetCommentsByPost(postID uuid.UUID) ([]models.Comment, error) {
	var comments []models.Comment
	err := s.db.Preload("User").Where("post_id = ?", postID).Order("created_at DESC").Find(&comments).Error
	return comments, err
}

func (s *CommentService) CreateComment(comment *models.Comment) error {
	return s.db.Create(comment).Error
}

func (s *CommentService) UpdateComment(comment *models.Comment) error {
	return s.db.Save(comment).Error
}

func (s *CommentService) DeleteComment(id uuid.UUID) error {
	return s.db.Delete(&models.Comment{}, "id = ?", id).Error
}
