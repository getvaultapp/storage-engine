package api

import (
	"database/sql"

	"github.com/gin-gonic/gin"
)

func CheckFileIntegrityHandler(c *gin.Context, db *sql.DB) {}

func GetStorageAnalyticsHandler(c *gin.Context, db *sql.DB) {}

func GetStorageInfoHandler(c *gin.Context, db *sql.DB) {}
