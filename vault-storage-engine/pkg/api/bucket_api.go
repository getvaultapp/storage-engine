package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/auth"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Bucket Handlers
func ListBucketsHandler(c *gin.Context) {

	// use ctx for outgping calls (how does that work?)

	db := c.MustGet("db").(*sql.DB)

	token, err := auth.GetTokenFromRequest(c)
	if err != nil {
		fmt.Printf("failed to get token from request, %v", err)
	}
	owner, err := auth.GetUsernameFromEmail(c, token)
	if err != nil {
		fmt.Printf("failed to owner_id from token, %v", err)
	}

	buckets, err := bucket.ListAllBuckets(db, owner)
	if err != nil {
		log.Printf("Error listing buckets: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list buckets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"buckets": buckets})
}

func CreateBucketHandler(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)

	var createRequest struct {
		BucketID string `json:"bucket_id" binding:"required"`
	}

	// Get the user email to get the username and append it automatically to owner section
	token, err := auth.GetTokenFromRequest(c)
	if err != nil {
		fmt.Printf("error getting token, %v", err)
	}

	owner, err := auth.GetUsernameFromEmail(c, token)
	if owner == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create bucket, user does not exists"})
		return
	}

	if err := c.ShouldBindJSON(&createRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err = bucket.CreateBucket(db, createRequest.BucketID, owner)

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bucket"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Bucket created successfully", "bucket_id": createRequest.BucketID})
}

func GetBucketHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	db := c.MustGet("db").(*sql.DB)

	token, err := auth.GetTokenFromRequest(c)
	if err != nil {
		fmt.Printf("failed to get token from request, %v", err)
	}

	authVerify, err := auth.VerifyBucketOwnership(c, db, bucketID, token)
	if !authVerify {
		fmt.Printf("verfying owner error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "access denied:"})
		return
	}
	bucket, err := bucket.GetBucket(db, bucketID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bucket not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bucket": bucket})
}

func DeleteBucketHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*config.Config)
	logger := c.MustGet("logger").(*zap.Logger)

	token, err := auth.GetTokenFromRequest(c)
	if err != nil {
		fmt.Printf("failed to get token from request, %v", err)
	}

	authVerify, err := auth.VerifyBucketOwnership(c, db, bucketID, token)
	if !authVerify {
		fmt.Printf("verfying owner error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "access denied:"})
		return
	}

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)

	err = datastorage.DeleteBucket(db, bucketID, store, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bucket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bucket deleted successfully", "bucket_id": bucketID})
}
