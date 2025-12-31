package services

import (
	"glib/demo/controllers/models"
	"time"
)

type PostSerivce struct {
	UserSerivce *UserSerivce
}

// @Provider singleton
func NewPostSerivce(userSerivce *UserSerivce) *PostSerivce {
	return &PostSerivce{
		UserSerivce: userSerivce,
	}
}

func (s *PostSerivce) GetPost(id int) *models.Post {
	return &models.Post{
		ID:        id,
		Title:     "Hello World",
		Body:      "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func (s *PostSerivce) GetPosts() []models.Post {
	return []models.Post{
		{
			ID:        1,
			Title:     "Hello World",
			Body:      "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        2,
			Title:     "Hello World",
			Body:      "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        3,
			Title:     "Hello World",
			Body:      "Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}
}
