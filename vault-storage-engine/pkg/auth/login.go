package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID       string
	Username string
	Password string
	Role     string
}

// NOTE: This is a dummy user database (replace with actual DB in production)
var users = map[string]User{
	"user1": {ID: "1", Username: "user1", Password: "$2a$10$7e/7QJm5x7QJm5x7QJm5x7e/7QJm5x7QJm5x7QJm5x7e/7QJm5x7QJm5x", Role: "user"},
	"admin": {ID: "2", Username: "admin", Password: "$2a$10$7e/7QJm5x7QJm5x7QJm5x7e/7QJm5x7QJm5x7QJm5x7e/7QJm5x7QJm5x", Role: "admin"},
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
	if !exists || bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password)) != nil {
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
