package internal

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
)

// GenerateSignature generates an HMAC signature for a URL.
func GenerateSignature(path string, expiration time.Time, secret string) string {
	message := fmt.Sprintf("%s:%d", path, expiration.Unix())
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifySignature verifies an HMAC signature.
func VerifySignature(path string, expirationStr, signature, secret string) (bool, error) {
	// Parse expiration
	expirationUnix, err := strconv.ParseInt(expirationStr, 10, 64)
	if err != nil {
		return false, err
	}

	expiration := time.Unix(expirationUnix, 0)

	// Check if expired
	if time.Now().After(expiration) {
		return false, nil
	}

	// Generate expected signature
	expected := GenerateSignature(path, expiration, secret)

	// Compare signatures (constant-time comparison)
	return hmac.Equal([]byte(signature), []byte(expected)), nil
}

// GenerateRandomString generates a random hex string of specified length.
func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
