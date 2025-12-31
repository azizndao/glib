package services

type UserSerivce struct{}

// @Provider singleton
func NewUserSerivce() *UserSerivce {
	return &UserSerivce{}
}

func (s *UserSerivce) Hello() string {
	return "Hello"
}
