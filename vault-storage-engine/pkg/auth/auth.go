package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// I need to get a jwtSecret key
var jwtSecret = []byte("super_secret_key")

// GenerateJWT creates a JWT for a user
func GenerateJWT(userID, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Expires in 24h
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
