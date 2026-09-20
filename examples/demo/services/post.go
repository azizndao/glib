package services

import (
	"context"
	"glib/demo/models"
	"strings"
	"uuid"

	"gorm.io/gorm"
)

type PostSerivce struct {
	db      *gorm.DB
	auditor Auditor
}

// @Provider singleton
func NewPostSerivce(db *gorm.DB, auditor *Auditor) *PostSerivce {
	return &PostSerivce{db: db, auditor: *auditor}
}

func (s *PostSerivce) GetPost(ctx context.Context, id uuid.UUID) (*models.Post, error) {
	return gorm.G[*models.Post](s.db).Preload("Author", nil).Preload("Comments", nil).First(ctx)
}

type PostPaginationParams struct {
	Page    int    `query:"page" validate:"required,min=1"`
	PerPage int    `query:"per_page" validate:"required,min=1,max=100"`
	Sort    string `query:"sort" validate:"omitempty,oneof=asc desc"`
}

func (s *PostSerivce) GetPosts(ctx context.Context, page int, perPage int) ([]models.Post, error) {
	return gorm.G[models.Post](s.db).
		Preload("Author", nil).
		Order("created_at DESC").
		Limit(perPage).
		Offset(perPage * (page - 1)).
		Find(ctx)
}

func (s *PostSerivce) SearchPosts(ctx context.Context, page, limit int, query *string, tags []string) ([]models.Post, error) {
	var posts []models.Post
	db := s.db.Preload("Author")

	if query != nil && *query != "" {
		searchTerm := "%" + *query + "%"
		db = db.Where("title LIKE ? OR body LIKE ?", searchTerm, searchTerm)
	}

	if len(tags) > 0 {
		for _, tag := range tags {
			db = db.Where("tags LIKE ?", "%"+tag+"%")
		}
	}

	db = db.Where("published = ?", true)

	if page > 0 && limit > 0 {
		offset := (page - 1) * limit
		db = db.Offset(offset).Limit(limit)
	}

	err := db.Order("created_at DESC").Find(&posts).Error
	return posts, err
}

func (s *PostSerivce) CreatePost(ctx context.Context, post *models.Post) error {
	if post.Slug == "" {
		post.Slug = generateSlug(post.Title)
	}
	return s.db.Create(post).Error
}

func (s *PostSerivce) UpdatePost(ctx context.Context, post *models.Post) error {
	return s.db.Save(post).Error
}

func (s *PostSerivce) DeletePost(ctx context.Context, id uuid.UUID) error {
	return s.db.Delete(&models.Post{}, "id = ?", id).Error
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	allowedChars := "abcdefghijklmnopqrstuvwxyz0123456789-"
	var result strings.Builder
	for _, char := range slug {
		if strings.ContainsRune(allowedChars, char) {
			result.WriteString(string(char))
		}
	}
	return result.String()
}
