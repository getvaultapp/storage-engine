package bucket

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// Object represents a stored file
type Object struct {
	ID            string
	BucketID      string
	LatestVersion string
}

// VersionMetadata represents the metadata for a version
type VersionMetadata struct {
	Data           []byte            `json:"data"`
	ShardLocations map[string]string `json:"shard_locations"`
	Proofs         map[string]string `json:"proofs"`
}

// AddObject adds an object to the database if it doesn't already exist
func AddObject(db *sql.DB, bucketID, objectID string) error {
	// Check if the object ID already exists
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM objects WHERE object_id = ? AND bucket_id = ?)"
	err := db.QueryRow(query, objectID, bucketID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check if object exists: %w", err)
	}

	if exists {
		// Object ID already exists, no need to add it again
		return nil
	}

	// Object ID doesn't exist, proceed to add it
	query = "INSERT INTO objects (bucket_id, object_id) VALUES (?, ?)"
	_, err = db.Exec(query, bucketID, objectID)
	if err != nil {
		return fmt.Errorf("failed to add object: %w", err)
	}

	return nil
}

// AddVersion inserts a new version for an object
/* func AddVersion(db *sql.DB, bucketID, objectID, versionID string, metadata VersionMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	query := `INSERT INTO versions (version_id, object_id, bucket_id, data, metadata) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(query, versionID, objectID, bucketID, metadata.Data, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to add version: %w", err)
	}

	// Update latest version for the object
	_, err = db.Exec(`UPDATE objects SET latest_version = ? WHERE object_id = ?`, versionID, objectID)
	if err != nil {
		return fmt.Errorf("failed to update object latest version: %w", err)
	}

	return nil
} */

// AddVersion inserts a new version for an object
func AddVersion(db *sql.DB, bucketID, objectID, versionID string, metadata VersionMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	query := `INSERT INTO versions (version_id, object_id, bucket_id, data, metadata) VALUES (?, ?, ?, ?, ?)`
	_, err = db.Exec(query, versionID, objectID, bucketID, metadata.Data, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to add version: %w", err)
	}

	// Update latest version for the object
	_, err = db.Exec(`UPDATE objects SET latest_version = ? WHERE object_id = ?`, versionID, objectID)
	if err != nil {
		return fmt.Errorf("failed to update object latest version: %w", err)
	}

	return nil
}

// GetObjectMetadata retrieves metadata for an object version
func GetObjectMetadata(db *sql.DB, objectID, versionID string) (*VersionMetadata, error) {
	query := `SELECT metadata FROM versions WHERE object_id = ? AND version_id = ?`
	row := db.QueryRow(query, objectID, versionID)

	var metadataJSON string
	err := row.Scan(&metadataJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("object version not found")
		}
		return nil, fmt.Errorf("failed to retrieve metadata: %w", err)
	}

	var metadata VersionMetadata
	err = json.Unmarshal([]byte(metadataJSON), &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to decode metadata: %w", err)
	}

	return &metadata, nil
}
