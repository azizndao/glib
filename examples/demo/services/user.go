package services

import (
	"context"
	"glib/demo/models"
	"uuid"

	"gorm.io/gorm"
)

type UserSerivce struct {
	db *gorm.DB
}

// @Provider singleton
func NewUserSerivce(db *gorm.DB) *UserSerivce {
	return &UserSerivce{db: db}
}

func (s *UserSerivce) users() gorm.Interface[models.User] {
	return gorm.G[models.User](s.db)
}

func (s *UserSerivce) GetUser(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.First(&user, "id = ?", id).Error
	return &user, err
}

func (s *UserSerivce) GetByUsername(ctx context.Context, username string) (models.User, error) {
	return s.users().Where(models.User{Username: username}).First(ctx)
}

func (s *UserSerivce) GetUsers() ([]models.User, error) {
	var users []models.User
	err := s.db.Find(&users).Error
	return users, err
}

func (s *UserSerivce) CreateUser(ctx context.Context, user *models.User) error {
	return s.users().Create(ctx, user)
}

func (s *UserSerivce) UpdateUser(ctx context.Context, id uuid.UUID, user *models.User) (*models.User, error) {
	if _, err := s.users().Where("id = ?", id).Updates(ctx, *user); err != nil {
		return nil, err
	}

	// Fetch the updated user
	updated, err := s.users().Where("id = ?", id).First(ctx)
	if err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s *UserSerivce) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := s.users().Where("id = ?", id).Delete(ctx)
	return err
}
