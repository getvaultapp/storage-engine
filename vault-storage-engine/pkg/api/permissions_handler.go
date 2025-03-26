package api

import (
	"database/sql"
	"net/http"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/acl"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/gin-gonic/gin"
)

func SetBucketPermissionsHandler(c *gin.Context) {
	bucketID := c.Param("bucket_id")

	db := c.MustGet("db").(*sql.DB)

	var req struct {
		Read  []string `json:"read"`
		Write []string `json:"write"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	err := bucket.SetBucketPermissions(db, bucketID, req.Read, req.Write)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set permissions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions updated"})
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
