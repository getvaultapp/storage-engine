package api

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/auth"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/models"
	"github.com/gin-gonic/gin"
)

// Auth Handlers
func LoginHandler(c *gin.Context) {
	var loginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		log.Printf("Error binding login request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	user, err := models.GetUserByUsername(db, loginRequest.Username)
	if err != nil {
		log.Printf("User not found: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := user.CheckPassword(loginRequest.Password); err != nil {
		log.Printf("Invalid password for user: %s", loginRequest.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(user.Username, "user")
	if err != nil {
		log.Printf("Error generating token for user: %s, error: %v", loginRequest.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// RegisterHandler handles user registration
func RegisterHandler(c *gin.Context) {
	var registerRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	user := &models.User{
		Username: registerRequest.Username,
	}

	if err := user.HashPassword(registerRequest.Password); err != nil {
		log.Printf("Error hashing password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	if err := models.CreateUser(db, user); err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}
