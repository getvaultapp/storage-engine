package acl

import (
	"database/sql"
	"fmt"
	"sync"
)

var aclCache = struct {
	sync.RWMutex
	entries map[string]map[string]bool
}{entries: make(map[string]map[string]bool)}

// CreateGroup adds a new group
func CreateGroup(db *sql.DB, groupID, name string) error {
	query := `INSERT INTO groups (group_id, name) VALUES (?, ?)`
	_, err := db.Exec(query, groupID, name)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	return nil
}

// AddUserToGroup assigns a user to a group
func AddUserToGroup(db *sql.DB, userID, groupID string) error {
	query := `INSERT INTO user_groups (user_id, group_id) VALUES (?, ?)`
	_, err := db.Exec(query, userID, groupID)
	if err != nil {
		return fmt.Errorf("failed to add user to group: %w", err)
	}
	return nil
}

// AddGroupPermission grants a group access to a resource
func AddGroupPermission(db *sql.DB, resourceID, resourceType, groupID, permission string) error {
	query := `INSERT INTO acl_groups (resource_id, resource_type, group_id, permission) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, resourceID, resourceType, groupID, permission)
	if err != nil {
		return fmt.Errorf("failed to add group ACL: %w", err)
	}
	return nil
}

// CheckPermissionWithGroups checks direct & group-based permissions
func CheckPermissionWithGroups(db *sql.DB, resourceID, resourceType, userID, permission string) (bool, error) {
	// Check direct user permission
	userAllowed, err := CheckPermissionWithCache(db, resourceID, resourceType, userID, permission)
	if err != nil || userAllowed {
		return userAllowed, err
	}

	// Get user's groups
	query := `SELECT group_id FROM user_groups WHERE user_id = ?`
	rows, err := db.Query(query, userID)
	if err != nil {
		return false, fmt.Errorf("failed to fetch user groups: %w", err)
	}
	defer rows.Close()

	// Check if any group has the required permission
	for rows.Next() {
		var groupID string
		if err := rows.Scan(&groupID); err != nil {
			return false, err
		}

		groupQuery := `SELECT COUNT(*) FROM acl_groups WHERE resource_id = ? AND group_id = ? AND permission = ?`
		var count int
		err := db.QueryRow(groupQuery, resourceID, groupID, permission).Scan(&count)
		if err != nil {
			return false, fmt.Errorf("failed to check group ACL: %w", err)
		}
		if count > 0 {
			return true, nil
		}
	}

	return false, nil
}
