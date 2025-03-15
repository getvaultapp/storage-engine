package api

import (
	"database/sql"
	"net/http"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/gin-gonic/gin"
)

// GetBucketHandler handles fetching a bucket's details
func GetBucketHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bucketID := c.Param("bucket_id")
		b, err := bucket.GetBucket(db, bucketID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bucket"})
			return
		}
		c.JSON(http.StatusOK, b)
	}
}

// CreateBucketHandler handles creating a new bucket
func CreateBucketHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var b bucket.Bucket
		if err := c.ShouldBindJSON(&b); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		// Call CreateBucket with the required arguments
		if err := bucket.CreateBucket(db, b.ID, b.Owner); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bucket"})
			return
		}
		c.JSON(http.StatusCreated, b)
	}
}
