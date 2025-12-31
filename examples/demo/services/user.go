package services

import "glib/demo/models"

type UserSerivce struct{}

// @Provider singleton
func NewUserSerivce() *UserSerivce {
	return &UserSerivce{}
}

func (s *UserSerivce) Hello() string {
	return "Hello"
}

func (s *UserSerivce) GetUser(id int) *models.User {
	return &models.User{
		ID:        id,
		FirstName: "John",
		LastName:  "Doe",
	}
}
