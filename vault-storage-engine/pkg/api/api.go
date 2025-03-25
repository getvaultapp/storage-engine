package api

import (
	"database/sql"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/auth"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRouter(db *sql.DB, cfg *config.Config, logger *zap.Logger) *gin.Engine {
	router := gin.Default()

	// Inject the database, config, and logger into the context
	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("config", cfg)
		c.Set("logger", logger)
		c.Next()
	})

	// Public endpoints
	router.POST("/auth/login", LoginHandler)
	router.POST("/auth/register", RegisterHandler)

	// Protected endpoints
	authGroup := router.Group("/v1")
	authGroup.Use(auth.JWTMiddleware())
	{
		authGroup.GET("/buckets", ListBucketsHandler)
		authGroup.POST("/buckets", CreateBucketHandler)
		authGroup.GET("/buckets/:bucketID", GetBucketHandler)
		authGroup.DELETE("/buckets/:bucketID", DeleteBucketHandler)

		authGroup.GET("/objects/:bucketID", ListObjectsHandler)
		authGroup.POST("/objects/:bucketID", UploadObjectHandler)
		authGroup.GET("/objects/:bucketID/:objectID", GetObjectHandler)
		authGroup.GET("/objects/:bucketID/:objectID/:versionID", GetObjectByVersionHandler)
		authGroup.POST("/objects/:bucketID/:objectID/update", UpdateObjectVersionHandler)
		authGroup.DELETE("/objects/:bucketID/:objectID/:versionID", DeleteObjectByVersionHandler)
		authGroup.DELETE("/objects/:bucketID/:objectID", DeleteObjectHandler)

		authGroup.GET("/objects/:bucketID/:objectID/version/:versionID/integrity", CheckFileIntegrityHandler)
		authGroup.GET("/objects/:bucketID/:objectID/analytics", GetStorageAnalyticsHandler)
		authGroup.GET("/objects/:bucketID/:objectID/info", GetStorageInfoHandler)
	}

	// ACL endpoints
	aclGroup := router.Group("/acl")
	aclGroup.Use(auth.JWTMiddleware())
	{
		aclGroup.POST("/permissions", AddPermissionHandler)
		aclGroup.GET("/permissions", ListPermissionsHandler)
		aclGroup.POST("/permissions/:objectID", AddObjectPermissionHandler)
		aclGroup.POST("/groups", CreateGroupHandler)
		aclGroup.POST("/group/:groupID", GrantGroupAccessHandler)
		aclGroup.POST("/groups/:groupID/users", AddUserToGroupHandler)
	}

	return router
}
