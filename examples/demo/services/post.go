package services

import (
	"glib/demo/models"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostSerivce struct {
	db *gorm.DB
}

// @Provider singleton
func NewPostSerivce(db *gorm.DB) *PostSerivce {
	return &PostSerivce{db: db}
}

func (s *PostSerivce) GetPost(id uuid.UUID) (*models.Post, error) {
	var post models.Post
	err := s.db.Preload("Author").Preload("Comments").First(&post, "id = ?", id).Error
	return &post, err
}

func (s *PostSerivce) GetPosts() ([]models.Post, error) {
	var posts []models.Post
	err := s.db.Preload("Author").Order("created_at DESC").Find(&posts).Error
	return posts, err
}

func (s *PostSerivce) SearchPosts(page, limit int, query *string, tags []string) ([]models.Post, error) {
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

func (s *PostSerivce) CreatePost(post *models.Post) error {
	if post.Slug == "" {
		post.Slug = generateSlug(post.Title)
	}
	return s.db.Create(post).Error
}

func (s *PostSerivce) UpdatePost(post *models.Post) error {
	return s.db.Save(post).Error
}

func (s *PostSerivce) DeletePost(id uuid.UUID) error {
	return s.db.Delete(&models.Post{}, "id = ?", id).Error
}

func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	allowedChars := "abcdefghijklmnopqrstuvwxyz0123456789-"
	result := ""
	for _, char := range slug {
		if strings.ContainsRune(allowedChars, char) {
			result += string(char)
		}
	}
	return result
}
