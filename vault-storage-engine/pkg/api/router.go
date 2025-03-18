package api

/* import (
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
		CreateBucketHandler(c, db)
	})
	r.GET("/buckets/:bucket_id", auth.JWTMiddleware(), auth.RBACMiddleware("reader"), func(c *gin.Context) {
		GetBucketHandler(c, db)
	})
	r.POST("/buckets/:bucket_id/objects", auth.JWTMiddleware(), auth.RBACMiddleware("writer"), func(c *gin.Context) {
		UploadObjectHandler(c, db)
	})
	r.GET("/buckets/:bucket_id/objects/:object_id", auth.JWTMiddleware(), auth.RBACMiddleware("reader"), func(c *gin.Context) {
		RetrieveObjectHandler(c, db)
	})
	r.GET("/buckets/:bucket_id/objects/:object_id/versions", auth.JWTMiddleware(), auth.RBACMiddleware("reader"), func(c *gin.Context) {
		ListVersionsHandler(c, db)
	})
	r.GET("/buckets/:bucket_id/objects/:object_id/versions/:version_id", auth.JWTMiddleware(), auth.RBACMiddleware("reader"), func(c *gin.Context) {
		RetrieveVersionHandler(c, db)
	})
	r.POST("/buckets/:bucket_id/permissions", auth.JWTMiddleware(), auth.RBACMiddleware("owner"), func(c *gin.Context) {
		SetBucketPermissionsHandler(c, db)
	})
	return r
} */
