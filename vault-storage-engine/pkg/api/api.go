package api

import (
	"database/sql"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/acl"
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

	// Check server
	router.POST("/", HomeHandler)

	// Protected endpoints
	authGroup := router.Group("/api")
	authGroup.Use(auth.JWTMiddleware())
	{
		authGroup.GET("/list/buckets", ListBucketsHandler)
		authGroup.POST("/create/buckets", CreateBucketHandler)
		authGroup.GET("/buckets/:bucketID", GetBucketHandler)
		authGroup.DELETE("/delete/buckets/:bucketID", DeleteBucketHandler)

		authGroup.GET("/list/objects/:bucketID", ListObjectsHandler)
		authGroup.POST("/upload/objects/:bucketID", UploadObjectHandler)
		authGroup.GET("/download/objects/:bucketID/:objectID", GetObjectHandler)
		authGroup.GET("/download/objects/:bucketID/:objectID/:versionID", GetObjectByVersionHandler)
		authGroup.POST("/update/objects/:bucketID/:objectID/update", UpdateObjectVersionHandler)
		authGroup.DELETE("/delete/objects/:bucketID/:objectID/:versionID", DeleteObjectByVersionHandler)
		authGroup.DELETE("/delete/objects/:bucketID/:objectID", DeleteObjectHandler)

		/* authGroup.GET("/objects/:bucketID/:objectID/:versionID", CheckFileIntegrityHandler)
		authGroup.GET("/objects/:bucketID/:objectID", GetStorageAnalyticsHandler)
		authGroup.GET("/objects/:bucketID/:objectID", GetStorageInfoHandler) */
	}

	// ACL and RBAC protected endpoints
	aclGroup := router.Group("/acl")
	aclGroup.Use(auth.JWTMiddleware(), acl.RBACMiddleware("admin"))
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
