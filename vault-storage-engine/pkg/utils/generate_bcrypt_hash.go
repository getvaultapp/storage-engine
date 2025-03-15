package utils

import (
	"log"

	"golang.org/x/crypto/bcrypt"
)

// GetBcrypt generates a bcrypt hash of the given password
func GetBcrypt(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Error generating bcrypt hash: %v", err)
	}
	return string(hash)
}
