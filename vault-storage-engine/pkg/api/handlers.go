package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/acl"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/auth"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/datastorage"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/sharding"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func HomeHandler(c *gin.Context) {
	log.Printf("Vault")
	c.JSON(http.StatusBadRequest, gin.H{"message": "Server running"})
}

// Auth Handlers
func LoginHandler(c *gin.Context) {
	// This is a sample handler
	var loginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		log.Printf("Error binding login request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Authenticate user (this is a placeholder, replace it with actual authentication logic)
	if loginRequest.Username != "admin" || loginRequest.Password != "password" {
		log.Printf("Invalid credentials for user: %s", loginRequest.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token, err := auth.GenerateJWT(loginRequest.Username, "admin")
	if err != nil {
		log.Printf("Error generating token for user: %s, error: %v", loginRequest.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token})
}

func RegisterHandler(c *gin.Context) {
	// We would implement registration here
	log.Printf("User registered successfully")
	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

// Bucket Handlers
func ListBucketsHandler(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)

	buckets, err := bucket.ListAllBuckets(db)
	if err != nil {
		log.Printf("Error listing buckets: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list buckets"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"buckets": buckets})
}

func CreateBucketHandler(c *gin.Context) {
	var createRequest struct {
		BucketID string `json:"bucket_id" binding:"required"`
		OwnerID  string `json:"owner_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&createRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)

	err := bucket.CreateBucket(db, createRequest.BucketID, createRequest.OwnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bucket"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Bucket created successfully", "bucket_id": createRequest.BucketID})
}

func GetBucketHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	db := c.MustGet("db").(*sql.DB)
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
	cfg := c.MustGet("config").(*viper.Viper)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg.GetString("shardStoreBasePath"))

	err := datastorage.DeleteBucket(db, bucketID, store, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bucket"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Bucket deleted successfully", "bucket_id": bucketID})
}

// Object Handlers
func ListObjectsHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	db := c.MustGet("db").(*sql.DB)

	objects, err := datastorage.ListObjects(db, bucketID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list objects"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"bucket_id": bucketID, "objects": objects})
}

func UploadObjectHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")

	var uploadRequest struct {
		ObjectName string `json:"object_name" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&uploadRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	cfg_value := c.MustGet("config").(*viper.Viper)
	cfg := utils.ConvertViperToConfig(cfg_value)
	logger := c.MustGet("logger").(*zap.Logger)

	data := []byte(uploadRequest.Content)
	store := sharding.NewLocalShardStore(cfg_value.GetString("shardStoreBasePath"))
	locations := cfg_value.GetStringSlice("shardLocations")
	objectID := uuid.New().String()

	_, shardLocations, proofs, err := datastorage.StoreData(db, data, bucketID, objectID, uploadRequest.ObjectName, store, cfg, locations, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store object"})
		return
	}

	proofsMap := utils.ConvertSliceToMap(proofs)

	c.JSON(http.StatusCreated, gin.H{
		"message":         "Object uploaded successfully",
		"bucket_id":       bucketID,
		"object_id":       objectID,
		"object_name":     uploadRequest.ObjectName,
		"shard_locations": shardLocations,
		"proofs":          proofsMap,
	})
}

func GetObjectHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")

	db := c.MustGet("db").(*sql.DB)
	cfg_value := c.MustGet("config").(*viper.Viper)
	cfg := utils.ConvertViperToConfig(cfg_value)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg_value.GetString("shardStoreBasePath"))
	data, filename, err := datastorage.RetrieveData(db, bucketID, objectID, "latest", store, cfg, logger)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}

	tmpFile, err := os.CreateTemp("", "object-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file"})
		return
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write data to temporary file"})
		return
	}

	c.FileAttachment(tmpFile.Name(), filename)
}

func GetObjectByVersionHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")
	versionID := c.Param("versionID")

	db := c.MustGet("db").(*sql.DB)
	cfg_value := c.MustGet("config").(*viper.Viper)
	cfg := utils.ConvertViperToConfig(cfg_value)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg_value.GetString("shardStoreBasePath"))
	data, filename, err := datastorage.RetrieveData(db, bucketID, objectID, versionID, store, cfg, logger)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Object not found"})
		return
	}

	tmpFile, err := os.CreateTemp("", "object-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary file"})
		return
	}
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write data to temporary file"})
		return
	}

	c.FileAttachment(tmpFile.Name(), filename)
}

func UpdateObjectVersionHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")

	var updateRequest struct {
		Filename string `json:"filename" binding:"required"`
	}

	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	cfg_value := c.MustGet("config").(*viper.Viper)
	cfg := utils.ConvertViperToConfig(cfg_value)
	logger := c.MustGet("logger").(*zap.Logger)

	getfile, versionID, err := bucket.UpdateFileVersionIfItExists(db, updateRequest.Filename, bucketID, objectID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check if file exists"})
		return
	}

	if updateRequest.Filename != getfile {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	data, err := os.ReadFile(updateRequest.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	store := sharding.NewLocalShardStore(cfg_value.GetString("shardStoreBasePath"))
	locations := cfg_value.GetStringSlice("shardLocations")

	_, _, _, err = datastorage.StoreDataWithVersion(db, data, bucketID, objectID, versionID, filepath.Base(updateRequest.Filename), store, cfg, locations, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store updated object"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Object updated successfully", "bucket_id": bucketID, "object_id": objectID})
}

func DeleteObjectHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*viper.Viper)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg.GetString("shardStoreBasePath"))
	err := datastorage.DeleteObject(db, bucketID, objectID, "all", store, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete object"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Object deleted successfully", "bucket_id": bucketID, "object_id": objectID})
}

func DeleteObjectByVersionHandler(c *gin.Context) {
	bucketID := c.Param("bucketID")
	objectID := c.Param("objectID")
	versionID := c.Param("versionID")

	db := c.MustGet("db").(*sql.DB)
	cfg := c.MustGet("config").(*viper.Viper)
	logger := c.MustGet("logger").(*zap.Logger)

	store := sharding.NewLocalShardStore(cfg.GetString("shardStoreBasePath"))
	err := datastorage.DeleteObjectByVersion(db, bucketID, objectID, versionID, store, logger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete object version"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Object version deleted successfully", "bucket_id": bucketID, "object_id": objectID, "version_id": versionID})
}

// ACL Handlers
func AddPermissionHandler(c *gin.Context) {
	var permissionRequest struct {
		ResourceID   string `json:"resource_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"`
		UserID       string `json:"user_id" binding:"required"`
		Permission   string `json:"permission" binding:"required"`
	}

	if err := c.ShouldBindJSON(&permissionRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	err := acl.AddPermission(db, permissionRequest.ResourceID, permissionRequest.ResourceType, permissionRequest.UserID, permissionRequest.Permission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add permission"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Permission added successfully"})
}

func ListPermissionsHandler(c *gin.Context) {
	db := c.MustGet("db").(*sql.DB)

	permissions, err := acl.ListPermissions(db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"permissions": permissions})
}

func AddObjectPermissionHandler(c *gin.Context) {
	var permissionRequest struct {
		ResourceID   string `json:"resource_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"`
		UserID       string `json:"user_id" binding:"required"`
		Permission   string `json:"permission" binding:"required"`
	}

	if err := c.ShouldBindJSON(&permissionRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	err := acl.AddPermission(db, permissionRequest.ResourceID, permissionRequest.ResourceType, permissionRequest.UserID, permissionRequest.Permission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add permission"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Permission added successfully"})
}

func CreateGroupHandler(c *gin.Context) {
	var groupRequest struct {
		GroupID string `json:"group_id" binding:"required"`
		Name    string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&groupRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	err := acl.CreateGroup(db, groupRequest.GroupID, groupRequest.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Group created successfully"})
}

func GrantGroupAccessHandler(c *gin.Context) {
	groupID := c.Param("groupID")

	var accessRequest struct {
		ResourceID   string `json:"resource_id" binding:"required"`
		ResourceType string `json:"resource_type" binding:"required"`
		Permission   string `json:"permission" binding:"required"`
	}

	if err := c.ShouldBindJSON(&accessRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	err := acl.AddGroupPermission(db, accessRequest.ResourceID, accessRequest.ResourceType, groupID, accessRequest.Permission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant group access"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Group access granted successfully"})
}

func AddUserToGroupHandler(c *gin.Context) {
	groupID := c.Param("groupID")

	var userRequest struct {
		UserID string `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&userRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	db := c.MustGet("db").(*sql.DB)
	err := acl.AddUserToGroup(db, userRequest.UserID, groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to group"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User added to group successfully"})
}

func CheckFileIntegrityHandler(c *gin.Context) {}

func GetStorageAnalyticsHandler(c *gin.Context) {}

func GetStorageInfoHandler(c *gin.Context) {}
