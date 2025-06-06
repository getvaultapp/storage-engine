package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

// JWTMiddleware validates JWT and extracts claims
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			c.Abort()
			return
		}

		// Parse and validate token
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set("username", claims["username"])
		c.Set("email", claims["email"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

// GetEmailFromToken extracts the email from a JWT token
func GetEmailFromToken(tokenString string) (string, error) {
	// Parse the JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(viper.GetString("jwtSecret")), nil
	})
	if err != nil {
		return "", fmt.Errorf("error parsing token: %v", err)
	}

	// Extract the email from the token claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}
	email, ok := claims["email"].(string)
	if !ok {
		return "", errors.New("email not found in token")
	}

	return email, nil
}

func VerifyBucketOwnership(c *gin.Context, db *sql.DB, bucketID, tokenString string) (bool, error) {
	// This should get the email from the token
	email, err := GetEmailFromToken(tokenString)
	if err != nil {
		return false, fmt.Errorf("failed to get email from token, %v", err)
	}

	// This should get the username from the email
	username, err := GetUsernameFromEmail(c, email)
	if err != nil {
		return false, fmt.Errorf("unable to get username from token: %w", err)
	}

	var owner string
	query := "SELECT owner FROM buckets WHERE bucket_id = ?"
	err = db.QueryRow(query, bucketID).Scan(&owner)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("bucket not found: %w", err)
		}
		return false, fmt.Errorf("failed to query bucket owner: %w", err)
	}

	if owner != username {
		return false, fmt.Errorf("access denied")
	}

	return true, nil
}

func GetTokenFromRequest(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is missing")
	}

	parts := strings.Fields(authHeader)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

func GetUsernameFromEmail(c *gin.Context, email string) (string, error) {
	db := c.MustGet("db").(*sql.DB)

	// I have a variable (string known as email), how do i query my DB to get the associated username with that email, they are both in table users
	query := `SELECT username FROM users WHERE email = ?`
	var username string
	err := db.QueryRow(query, email).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no user found with email: %v", email)
		}
		return "", fmt.Errorf("failed to query database, %v", err)
	}

	return username, nil
}
