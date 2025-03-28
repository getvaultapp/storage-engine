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

		for _, claim := range claims {
			fmt.Println(claim)
		}

		// Store claims in context
		c.Set("useranme", claims["username"])
		c.Set("email", claims["email"])
		c.Set("role", claims["role"])
		c.Next()
	}
}

/* func GetUsernameFromToken(tokenString, jwtSecret string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return "", fmt.Errorf("error parsing token: %v", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", fmt.Errorf("username not found in token claims")
		}
		return username, nil
	}

	return "", fmt.Errorf("invalid token")
} */

// GetUsernameFromToken extracts the username from a JWT token
func GetEmailFromToken(tokenString string) (string, error) {
	// Parse the JWT token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Assuming you're using a secret key to sign the tokens
		return []byte(viper.GetString("jwtSecret")), nil
	})
	if err != nil {
		return "", fmt.Errorf("%v", err)
	}

	// Extract the username from the token claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}
	email, ok := claims["email"].(string)
	if !ok {
		return "", errors.New("username not found in token")
	}

	return email, nil
}

func VerifyBucketOwnership(c *gin.Context, db *sql.DB, bucketID, tokenString string) (bool, error) {
	// This should get the email from the token
	email, err := GetEmailFromToken(tokenString)
	println(email)
	if err != nil {
		return false, fmt.Errorf("failed to get email from token, %v", err)
	}

	// This should get the username from the email
	username, err := GetUsernameFromEmail(c, email)
	println(username)
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

func GetUsernameFromEmail(c *gin.Context, token string) (string, error) {
	db := c.MustGet("db").(*sql.DB)
	email, err := GetEmailFromToken(token)
	if err != nil {
		fmt.Printf("failed to get useremail from token %v", err)
	}

	// I have a variable (string known as email), how do i query my DB to get the associated username with that email, they are both in table users
	query := `SELECT username FROM users WHERE email = ?`
	var username string
	err = db.QueryRow(query, email).Scan(&username)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("")
		}
		fmt.Printf("failed to query database, %v", err)
	}

	return username, nil
}
