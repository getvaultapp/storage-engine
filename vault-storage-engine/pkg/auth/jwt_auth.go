package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

var jwtSecret []byte

// Initialize JWT secret

func InitJWTSecret() {
	jwtSecret = []byte(viper.GetString("jwtSecret"))
}

// GenerateJWT creates a JWT for a user

func GenerateJWT(userID, Emai, role string) (string, error) {
	claims := jwt.MapClaims{
		"username": userID,
		"email":    Emai,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // Expires in 24h
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
