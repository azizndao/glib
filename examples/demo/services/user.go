package services

import (
	"glib/demo/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserSerivce struct {
	db *gorm.DB
}

// @Provider singleton
func NewUserSerivce(db *gorm.DB) *UserSerivce {
	return &UserSerivce{db: db}
}

func (s *UserSerivce) GetUser(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.First(&user, "id = ?", id).Error
	return &user, err
}

func (s *UserSerivce) GetUsers() ([]models.User, error) {
	var users []models.User
	err := s.db.Find(&users).Error
	return users, err
}

func (s *UserSerivce) CreateUser(user *models.User) error {
	return s.db.Create(user).Error
}

func (s *UserSerivce) UpdateUser(id uuid.UUID, user *models.User) (*models.User, error) {
	// Use Updates with a map to update specific fields
	updates := map[string]any{
		"first_name": user.FirstName,
		"last_name":  user.LastName,
		"bio":        user.Bio,
		"active":     user.Active,
	}

	if err := s.db.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	// Fetch the updated user
	var updated models.User
	if err := s.db.First(&updated, "id = ?", id).Error; err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s *UserSerivce) DeleteUser(id uuid.UUID) error {
	return s.db.Delete(&models.User{}, "id = ?", id).Error
}
