package utils

import (
	"golang.org/x/crypto/bcrypt"
)

func GetBcrypt(text string) (string, error) {
	// Generate the bcrypt hash
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Print the hash
	return string(hash), nil
}
