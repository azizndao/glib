package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/azizndao/glib"
	"github.com/azizndao/glib/common/errors"
	"github.com/google/uuid"
)

// JWT claims structure
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	Exp    int64     `json:"exp"`
}

// JWTSecret is the secret key for signing tokens (in production, use env variable)
var JWTSecret = []byte("your-secret-key-change-in-production")

// CreateToken generates a JWT token for a user
func CreateToken(userID uuid.UUID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Exp:    time.Now().Add(24 * time.Hour).Unix(),
	}

	// Create header and payload
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// Create signature
	message := header + "." + payload
	h := hmac.New(sha256.New, JWTSecret)
	h.Write([]byte(message))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return message + "." + signature, nil
}

// VerifyToken validates a JWT token and returns the claims
func VerifyToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.Unauthorized("Invalid token format", nil)
	}

	// Verify signature
	message := parts[0] + "." + parts[1]
	h := hmac.New(sha256.New, JWTSecret)
	h.Write([]byte(message))
	expectedSignature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if parts[2] != expectedSignature {
		return nil, errors.Unauthorized("Invalid token signature", nil)
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.Unauthorized("Invalid token payload", err)
	}

	var claims Claims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, errors.Unauthorized("Invalid token claims", err)
	}

	// Check expiration
	if time.Now().Unix() > claims.Exp {
		return nil, errors.Unauthorized("Token expired", nil)
	}

	return &claims, nil
}

// Auth middleware verifies JWT token and sets user context
func Auth(next glib.HandleFunc) glib.HandleFunc {
	return func(c *glib.Ctx) error {
		// Get authorization header
		authHeader := c.Authorization()
		if authHeader == "" {
			return errors.Unauthorized("Missing authorization header", nil)
		}

		// Extract token (Bearer <token>)
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return errors.Unauthorized("Invalid authorization format", nil)
		}

		token := parts[1]

		// Verify token
		claims, err := VerifyToken(token)
		if err != nil {
			return err
		}

		// Set user ID in context
		c.SetValue("user_id", claims.UserID)
		c.SetValue("user_email", claims.Email)

		return next(c)
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *glib.Ctx) (uuid.UUID, error) {
	userID, ok := c.GetValue("user_id").(uuid.UUID)
	if !ok {
		return uuid.Nil, errors.Unauthorized("User not authenticated", nil)
	}
	return userID, nil
}
