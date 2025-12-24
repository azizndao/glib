package controllers

import (
	"crypto/sha256"
	"encoding/hex"
	"glib/example/fullstack/middleware"
	"glib/example/fullstack/models"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/common/container"
	"github.com/azizndao/glib/common/errors"
	"github.com/azizndao/glib/database"
	"github.com/azizndao/glib/foundation"
	"gorm.io/gorm"
)

// AuthController handles authentication operations.
type AuthController struct {
	db *gorm.DB
}

func NewAuth(app *foundation.Application) *AuthController {
	dbManager, _ := container.Resolve[*database.Manager](app.Container())
	conn, _ := dbManager.DB()
	return &AuthController{db: conn.DB()}
}

// RegisterRequest represents the registration request payload.
type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

// LoginRequest represents the login request payload.
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents the login/register response.
type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// Register handles user registration.
// POST /auth/register
func (ctrl *AuthController) Register(c *glib.Ctx) error {
	var req RegisterRequest
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	// Check if user exists
	var existingUser models.User
	if err := ctrl.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return errors.Conflict("Email already registered", nil)
	}

	// Hash password (simplified for demo)
	hash := sha256.Sum256([]byte(req.Password))
	hashedPassword := hex.EncodeToString(hash[:])

	// Create user
	user := models.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: hashedPassword,
	}

	if err := ctrl.db.Create(&user).Error; err != nil {
		return errors.InternalServerError("Failed to create user", err)
	}

	// Generate token
	token, err := middleware.CreateToken(user.ID, user.Email)
	if err != nil {
		return errors.InternalServerError("Failed to generate token", err)
	}

	return c.Status(201).JSON(LoginResponse{
		Token: token,
		User:  &user,
	})
}

// Login handles user authentication.
// POST /auth/login
func (ctrl *AuthController) Login(c *glib.Ctx) error {
	var req LoginRequest
	if err := c.ValidateBody(&req); err != nil {
		return err
	}

	// Find user
	var user models.User
	if err := ctrl.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		return errors.Unauthorized("Invalid credentials", nil)
	}

	// Verify password
	hash := sha256.Sum256([]byte(req.Password))
	hashedPassword := hex.EncodeToString(hash[:])

	if user.Password != hashedPassword {
		return errors.Unauthorized("Invalid credentials", nil)
	}

	// Generate token
	token, err := middleware.CreateToken(user.ID, user.Email)
	if err != nil {
		return errors.InternalServerError("Failed to generate token", err)
	}

	return c.JSON(LoginResponse{
		Token: token,
		User:  &user,
	})
}
