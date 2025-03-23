package acl

import (
	"database/sql"
	"fmt"
)

type Permission struct {
	ResourceID   string
	ResourceType string
	UserID       string
	Permission   string
}

// AddPermission grants a user read/write access to a bucket or object
func AddPermission(db *sql.DB, resourceID, resourceType, userID, permission string) error {
	query := `INSERT INTO acl (resource_id, resource_type, user_id, permission) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, resourceID, resourceType, userID, permission)
	if err != nil {
		return fmt.Errorf("failed to add ACL entry: %w", err)
	}
	return nil
}

// CheckPermission verifies if a user has access to a bucket/object
func CheckPermission(db *sql.DB, resourceID, userID, permission string) (bool, error) {
	query := `SELECT COUNT(*) FROM acl WHERE resource_id = ? AND user_id = ? AND permission = ?`
	var count int
	err := db.QueryRow(query, resourceID, userID, permission).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check ACL: %w", err)
	}
	return count > 0, nil
}

// CheckPermissionWithInheritance verifies user access, including inherited roles
func CheckPermissionWithInheritance(db *sql.DB, resourceID, resourceType, userID, permission string) (bool, error) {
	// Check direct permission
	query := `SELECT COUNT(*) FROM acl WHERE resource_id = ? AND user_id = ? AND permission = ?`
	var count int
	err := db.QueryRow(query, resourceID, userID, permission).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check ACL: %w", err)
	}
	if count > 0 {
		return true, nil
	}

	// If checking an object, check inherited bucket permissions
	if resourceType == "object" {
		bucketQuery := `SELECT bucket_id FROM objects WHERE object_id = ?`
		var bucketID string
		err := db.QueryRow(bucketQuery, resourceID).Scan(&bucketID)
		if err != nil {
			return false, fmt.Errorf("failed to get bucket for object: %w", err)
		}

		// Check if user has bucket-level access
		err = db.QueryRow(query, bucketID, userID, permission).Scan(&count)
		if err != nil {
			return false, fmt.Errorf("failed to check inherited bucket ACL: %w", err)
		}
		if count > 0 {
			return true, nil
		}
	}

	return false, nil
}

// CachePermission stores a permission check result
func CachePermission(userID, resourceID, permission string, allowed bool) {
	aclCache.Lock()
	defer aclCache.Unlock()

	if _, exists := aclCache.entries[userID]; !exists {
		aclCache.entries[userID] = make(map[string]bool)
	}
	aclCache.entries[userID][resourceID+":"+permission] = allowed
}

// GetCachedPermission checks if a permission is cached
func GetCachedPermission(userID, resourceID, permission string) (bool, bool) {
	aclCache.RLock()
	defer aclCache.RUnlock()

	if userPerms, exists := aclCache.entries[userID]; exists {
		result, found := userPerms[resourceID+":"+permission]
		return result, found
	}
	return false, false
}

// CheckPermissionWithCache checks direct & group-based permissions using cache
func CheckPermissionWithCache(db *sql.DB, resourceID, resourceType, userID, permission string) (bool, error) {
	// Check cache first
	if cached, found := GetCachedPermission(userID, resourceID, permission); found {
		return cached, nil
	}

	// Check permission in database
	allowed, err := CheckPermissionWithInheritance(db, resourceID, resourceType, userID, permission)
	if err != nil {
		return false, err
	}

	// Cache result
	CachePermission(userID, resourceID, permission, allowed)
	return allowed, nil
}

func ListPermissions(db *sql.DB) ([]Permission, error) {
	query := "SELECT resource_id, resource_type, user_id, permission FROM permissions"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var permission Permission
		if err := rows.Scan(&permission.ResourceID, &permission.ResourceType, &permission.UserID, &permission.Permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}
