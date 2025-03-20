package api

import (
	"database/sql"
	"net/http"

	"github.com/getvaultapp/vault-storage-engine/pkg/acl"
	"github.com/gin-gonic/gin"
)

func CreateGroupHandler(c *gin.Context, db *sql.DB) {
	var req struct {
		GroupID string `json:"group_id"`
		Name    string `json:"name"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := acl.CreateGroup(db, req.GroupID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create group"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Group created", "group_id": req.GroupID})
}

func AddUserToGroupHandler(c *gin.Context, db *sql.DB) {
	var req struct {
		UserID  string `json:"user_id"`
		GroupID string `json:"group_id"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := acl.AddUserToGroup(db, req.UserID, req.GroupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add user to group"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User added to group"})
}

func GrantGroupAccessHandler(c *gin.Context, db *sql.DB) {
	var req struct {
		GroupID    string `json:"group_id"`
		Permission string `json:"permission"` // "read" or "write"
	}

	resourceID := c.Param("bucket_id") // Default to bucket, override for objects

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := acl.AddGroupPermission(db, resourceID, "bucket", req.GroupID, req.Permission)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant group access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group access granted"})
}
