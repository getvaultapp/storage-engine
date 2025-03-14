package api

import (
	"database/sql"

	"github.com/getvault-mvp/vault-base/pkg/auth"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// Initialize store, cfg, and logger
	cfg = config.LoadConfig()
	store = sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	logger, _ = zap.NewProduction()

	r.POST("/buckets", auth.JWTMiddleware(), auth.RBACMiddleware("owner"), func(c *gin.Context) {
		createBucketHandler(c, db)
	})
	r.GET("/buckets/:bucket_id", auth.JWTMiddleware(), auth.RBACMiddleware("reader"), func(c *gin.Context) {
		getBucketHandler(c, db)
	})
	r.POST("/buckets/:bucket_id/objects", auth.JWTMiddleware(), auth.RBACMiddleware("writer"), func(c *gin.Context) {
		uploadObjectHandler(c, db)
	})
	r.GET("/buckets/:bucket_id/objects/:object_id", auth.JWTMiddleware(), auth.RBACMiddleware("reader"), func(c *gin.Context) {
		retrieveObjectHandler(c, db)
	})
	r.GET("/buckets/:bucket_id/objects/:object_id/versions", auth.JWTMiddleware(), auth.RBACMiddleware("reader"), func(c *gin.Context) {
		listVersionsHandler(c, db)
	})
	r.GET("/buckets/:bucket_id/objects/:object_id/versions/:version_id", auth.JWTMiddleware(), auth.RBACMiddleware("reader"), func(c *gin.Context) {
		retrieveVersionHandler(c, db)
	})
	r.POST("/buckets/:bucket_id/permissions", auth.JWTMiddleware(), auth.RBACMiddleware("owner"), func(c *gin.Context) {
		setPermissionsHandler(c, db)
	})

	return r
}

/* package api

import (
	"database/sql"
	// "github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/gin-gonic/gin"
)

func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	r.POST("/buckets", func(c *gin.Context) {
		// Handler to create a new bucket
	})

	r.GET("/buckets/:bucket_id", func(c *gin.Context) {
		// Handler to get bucket details
	})

	r.POST("/buckets/:bucket_id/objects", func(c *gin.Context) {
		// Handler to store an object in a bucket
	})

	r.GET("/buckets/:bucket_id/objects/:object_id", func(c *gin.Context) {
		// Handler to get object details
	})

	r.GET("/buckets/:bucket_id/objects/:object_id/versions", func(c *gin.Context) {
		// Handler to get object versions
	})

	r.GET("/buckets/:bucket_id/objects/:object_id/versions/:version_id", func(c *gin.Context) {
		// Handler to get a specific version of an object
	})

	r.POST("/buckets/:bucket_id/permissions", func(c *gin.Context) {
		// Handler to set permissions for a bucket
	})

	return r
}
*/
