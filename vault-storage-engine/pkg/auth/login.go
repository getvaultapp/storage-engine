package auth

import (
	"net/http"

	"log"

	"github.com/getvault-mvp/vault-base/pkg/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string
	Username string
	Password string
	Role     string
}

var password string = "password"

// Dummy user database (replace with actual DB in production)
var users = map[string]User{
	"user1": {ID: "1", Username: "user1", Password: utils.GetBcrypt(password), Role: "user"},
	"admin": {ID: "2", Username: "admin", Password: utils.GetBcrypt(password), Role: "admin"},
}

// LoginHandler handles user login and generates a JWT token
func LoginHandler(c *gin.Context) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, exists := users[loginData.Username]
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Debug statements
	log.Printf("Stored hash for user %s: %s", user.Username, user.Password)
	log.Printf("Password being compared: %s", loginData.Password)

	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password))
	if err != nil {
		log.Printf("Error comparing passwords: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := GenerateJWT(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}
