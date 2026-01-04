package auth

import (
	"context"
	"glib/demo/models"
	"glib/demo/services"

	"github.com/azizndao/glib/errs"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// @Controller path=/api/v1/auth tags=public
type Controller struct {
	UserService *services.UserSerivce
	JWTService  *services.JWTService
	Auditor     *services.Auditor
}

// Helper function to convert models.User to UserResponse
func toUserResponse(user *models.User) *UserResponse {
	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Bio:       user.Bio,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

// @Route method=POST path=/register
func (c *Controller) Register(ctx context.Context, req RegisterRequest) (*UserResponse, error) {
	c.Auditor.LogAction("user registration attempt: " + req.Username)

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := &models.User{
		ID:           uuid.New(),
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Bio:          req.Bio,
		Active:       true,
	}

	err = c.UserService.CreateUser(newUser)
	if err != nil {
		return nil, err
	}

	c.Auditor.LogAction("user registered: " + newUser.Username)
	return toUserResponse(newUser), nil
}

// @Route method=POST path=/login
func (c *Controller) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	c.Auditor.LogAction("login attempt: " + req.Username)

	// Find user by username
	users, err := c.UserService.GetUsers()
	if err != nil {
		return nil, err
	}

	var foundUser *models.User
	for _, user := range users {
		if user.Username == req.Username {
			foundUser = &user
			break
		}
	}

	if foundUser == nil {
		return nil, errs.NewUnauthorized().WithMessage("invalid credentials")
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(foundUser.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errs.NewUnauthorized().WithMessage("invalid credentials")
	}

	if !foundUser.Active {
		return nil, errs.NewForbidden().WithMessage("account is inactive")
	}

	// Generate JWT token
	token, err := c.JWTService.GenerateToken(foundUser.ID, foundUser.Username, foundUser.Email)
	if err != nil {
		return nil, err
	}

	c.Auditor.LogAction("user logged in: " + foundUser.Username)

	response := &LoginResponse{
		User:  toUserResponse(foundUser),
		Token: token,
	}

	return response, nil
}

// @Route method=GET path=/me tags=protected
func (c *Controller) GetMe(ctx context.Context) (*UserResponse, error) {
	// TODO: Get user ID from context (set by auth middleware)
	// For now, return the first user as a demo (NOT PRODUCTION READY)

	users, err := c.UserService.GetUsers()
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, errs.NewUnauthorized().WithMessage("not authenticated")
	}

	// Return first user as demo
	return toUserResponse(&users[0]), nil
}

// @Route method=PUT path=/me tags=protected
func (c *Controller) UpdateProfile(ctx context.Context, req UpdateProfileRequest) (*UserResponse, error) {
	// TODO: Get user ID from context (set by auth middleware)
	// For now, update the first user as a demo (NOT PRODUCTION READY)

	users, err := c.UserService.GetUsers()
	if err != nil {
		return nil, err
	}

	if len(users) == 0 {
		return nil, errs.NewUnauthorized().WithMessage("not authenticated")
	}

	user := users[0]

	// Apply updates if provided
	if req.FirstName != nil {
		user.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		user.LastName = *req.LastName
	}
	if req.Bio != nil {
		user.Bio = *req.Bio
	}

	updatedUser, err := c.UserService.UpdateUser(user.ID, &user)
	if err != nil {
		return nil, err
	}

	c.Auditor.LogAction("profile updated: " + updatedUser.Username)
	return toUserResponse(updatedUser), nil
}

// @Route method=DELETE path=/logout tags=protected
func (c *Controller) Logout(ctx context.Context) error {
	c.Auditor.LogAction("user logout")
	// Token invalidation would go here (e.g., add to blacklist)
	// For stateless JWT, client just deletes the token
	return nil
}

// @Route method=GET path=/users/{id}
func (c *Controller) GetUser(ctx context.Context, id uuid.UUID) (*UserResponse, error) {
	user, err := c.UserService.GetUser(id)
	if err != nil {
		return nil, errs.NewNotFound().WithMessage("user not found")
	}
	return toUserResponse(user), nil
}
