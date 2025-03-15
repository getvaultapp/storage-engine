package api

import (
	"database/sql"

	"github.com/getvault-mvp/vault-base/pkg/auth"
	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/gin-gonic/gin"
)

// SetupRouter sets up the API routes
func SetupRouter(db *sql.DB) *gin.Engine {
	r := gin.Default()

	// Public routes
	r.POST("/login", auth.LoginHandler)

	// Protected routes
	protected := r.Group("/")
	protected.Use(auth.JWTMiddleware())

	// Bucket routes
	protected.GET("/buckets/:bucket_id", bucket.ACLMiddleware(db), GetBucketHandler(db))
	protected.POST("/buckets", CreateBucketHandler(db))

	return r
}
