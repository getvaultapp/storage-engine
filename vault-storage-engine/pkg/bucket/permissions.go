package bucket

import (
	"database/sql"
	"fmt"
)

// SetBucketPermissions sets read/write permissions for a bucket
func SetBucketPermissions(db *sql.DB, bucketID string, read, write []string) error {
	// Insert read permissions
	for _, userID := range read {
		err := addPermission(db, bucketID, "bucket", userID, "read")
		if err != nil {
			return fmt.Errorf("failed to set read permissions: %w", err)
		}
	}

	// Insert write permissions
	for _, userID := range write {
		err := addPermission(db, bucketID, "bucket", userID, "write")
		if err != nil {
			return fmt.Errorf("failed to set write permissions: %w", err)
		}
	}

	return nil
}

func addPermission(db *sql.DB, resourceID, resourceType, userID, permission string) error {
	query := `INSERT INTO acl (resource_id, resource_type, user_id, permission) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, resourceID, resourceType, userID, permission)
	if err != nil {
		return fmt.Errorf("failed to add ACL entry: %w", err)
	}
	return nil
}

// ListObjectVersions lists all versions of an object
func ListObjectVersions(db *sql.DB, objectID string) ([]string, error) {
	query := `SELECT version_id FROM versions WHERE object_id = ?`
	rows, err := db.Query(query, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list object versions: %w", err)
	}
	defer rows.Close()

	var versions []string
	for rows.Next() {
		var versionID string
		if err := rows.Scan(&versionID); err != nil {
			return nil, fmt.Errorf("failed to scan version ID: %w", err)
		}
		versions = append(versions, versionID)
	}
	return versions, nil
}
