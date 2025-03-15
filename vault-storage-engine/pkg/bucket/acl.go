package bucket

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CheckBucketAccess checks if the user has access to the bucket
func CheckBucketAccess(db *sql.DB, userID, bucketID string) error {
	var accessExists bool
	query := "SELECT EXISTS(SELECT 1 FROM acl WHERE resource_id = ? AND resource_type = 'bucket' AND user_id = ?)"
	err := db.QueryRow(query, bucketID, userID).Scan(&accessExists)
	if err != nil {
		return fmt.Errorf("failed to check bucket access: %w", err)
	}

	if !accessExists {
		return fmt.Errorf("access denied")
	}

	return nil
}

// ACLMiddleware checks if the user has access to the requested resource
func ACLMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		bucketID := c.Param("bucket_id")
		if err := CheckBucketAccess(db, userID.(string), bucketID); err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return
		}

		c.Next()
	}
}
