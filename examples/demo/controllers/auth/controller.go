package auth

import (
	"context"
	"glib/demo/models"
	"glib/demo/services"

	"github.com/azizndao/glib"
	"github.com/google/uuid"
)

// @Controller path=/api/v1/auth tags=public
type Controller struct {
	UserSerivce *services.UserSerivce
	Auditor     *services.Auditor // Transient provider with singleton dependency
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
func (c *Controller) Register(ctx context.Context, req RegisterRequest) glib.Result[*UserResponse] {
	c.Auditor.LogAction("user registration attempt")

	// TODO: Hash password properly (use bcrypt in production)
	// For now, we'll just store it as-is (NOT PRODUCTION READY)
	newUser := &models.User{
		ID:        uuid.New(),
		Email:     req.Email,
		Username:  req.Username,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Bio:       req.Bio,
		Active:    true,
	}

	err := c.UserSerivce.CreateUser(newUser)
	if err != nil {
		return glib.Fail[*UserResponse](err)
	}

	c.Auditor.LogAction("user registered: " + newUser.Username)
	return glib.Created(toUserResponse(newUser))
}

// @Route method=POST path=/login
func (c *Controller) Login(ctx context.Context, req LoginRequest) glib.Result[*LoginResponse] {
	c.Auditor.LogAction("login attempt: " + req.Username)

	// Find user by username
	users, err := c.UserSerivce.GetUsers()
	if err != nil {
		return glib.Fail[*LoginResponse](err)
	}

	var foundUser *models.User
	for _, user := range users {
		if user.Username == req.Username {
			foundUser = &user
			break
		}
	}

	if foundUser == nil {
		return glib.Unauthorized[*LoginResponse]("invalid credentials")
	}

	// TODO: Verify password hash (use bcrypt.CompareHashAndPassword in production)
	// For now, we're just checking if user exists (NOT PRODUCTION READY)

	if !foundUser.Active {
		return glib.Forbidden[*LoginResponse]("account is inactive")
	}

	c.Auditor.LogAction("user logged in: " + foundUser.Username)

	// TODO: Generate JWT token in production
	response := &LoginResponse{
		User:  toUserResponse(foundUser),
		Token: "dummy-token-" + foundUser.ID.String(), // Replace with real JWT
	}

	return glib.OK(response)
}

// @Route method=GET path=/me
func (c *Controller) GetMe(ctx context.Context) glib.Result[*UserResponse] {
	// TODO: Get user ID from JWT token/session in production
	// For now, return the first user as a demo (NOT PRODUCTION READY)

	users, err := c.UserSerivce.GetUsers()
	if err != nil {
		return glib.Fail[*UserResponse](err)
	}

	if len(users) == 0 {
		return glib.Unauthorized[*UserResponse]("not authenticated")
	}

	// Return first user as demo
	return glib.OK(toUserResponse(&users[0]))
}

// @Route method=PUT path=/me
func (c *Controller) UpdateProfile(ctx context.Context, req UpdateProfileRequest) glib.Result[*UserResponse] {
	// TODO: Get user ID from JWT token/session in production
	// For now, update the first user as a demo (NOT PRODUCTION READY)

	users, err := c.UserSerivce.GetUsers()
	if err != nil {
		return glib.Fail[*UserResponse](err)
	}

	if len(users) == 0 {
		return glib.Unauthorized[*UserResponse]("not authenticated")
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

	updatedUser, err := c.UserSerivce.UpdateUser(user.ID, &user)
	if err != nil {
		return glib.Fail[*UserResponse](err)
	}

	c.Auditor.LogAction("profile updated: " + updatedUser.Username)
	return glib.OK(toUserResponse(updatedUser))
}

// @Route method=DELETE path=/logout
func (c *Controller) Logout(ctx context.Context) glib.Result[any] {
	c.Auditor.LogAction("user logout")
	// TODO: Invalidate JWT token/session in production
	return glib.NoContent[any]()
}

// @Route method=GET path=/users/{id}
func (c *Controller) GetUser(ctx context.Context, id uuid.UUID) glib.Result[*UserResponse] {
	user, err := c.UserSerivce.GetUser(id)
	if err != nil {
		return glib.NotFound[*UserResponse]("user not found")
	}
	return glib.OK(toUserResponse(user))
}
