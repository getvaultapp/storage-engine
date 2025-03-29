package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/auth"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/models"
	"github.com/gin-gonic/gin"
)

// Auth Handlers
func LoginHandler(c *gin.Context) {
	var loginRequest struct {
		//Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		log.Printf("Error binding login request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	//user, err := models.GetUserByUsername(db, loginRequest.Username)
	user, err := models.GetUserByEmail(db, loginRequest.Email)
	if err != nil {
		log.Printf("User not found: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := user.CheckPassword(loginRequest.Password); err != nil {
		//log.Printf("Invalid password for user: %s", loginRequest.Username)
		log.Printf("Invalid password for user: %s", loginRequest.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(user.Username, user.Email, "user")
	if err != nil {
		log.Printf("Error generating token for user: %s, error: %v", loginRequest.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.SetCookie("token", token, 24*3600, "/", "", false, true) // Cookie set to exists for 24 hours
	fmt.Println("token saved as cookie")

	c.JSON(http.StatusOK, gin.H{"token": token})
}

// RegisterHandler handles user registration
func RegisterHandler(c *gin.Context) {
	var registerRequest struct {
		Username             string `json:"username" binding:"required"`
		Email                string `json:"email" binding:"required"`
		Password             string `json:"password" binding:"required"`
		PasswordConfirmation string `json:"password_confirmation" binding:"required"`
	}

	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: Binding failed"})
		return
	}

	if registerRequest.Password != registerRequest.PasswordConfirmation {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: Password does not match confirmation"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	user := &models.User{
		Username: registerRequest.Username,
		Email:    registerRequest.Email,
	}

	if err := user.HashPassword(registerRequest.Password); err != nil {
		log.Printf("Error hashing password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user, password"})
		return
	}

	if err := models.CreateUser(db, user); err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user, can't create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully"})
}
