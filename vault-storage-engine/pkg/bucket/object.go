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

// VersionMetadata represents metadata for an object version
type VersionMetadata struct {
	ShardLocations map[string]string `json:"shards"` // Maps node_id -> shard_hash
	Proofs         []string          `json:"proofs"`
}

// AddObject inserts a new object
func AddObject(db *sql.DB, bucketID, objectID string) error {
	query := `INSERT INTO objects (object_id, bucket_id) VALUES (?, ?)`
	_, err := db.Exec(query, objectID, bucketID)
	if err != nil {
		return fmt.Errorf("failed to add object: %w", err)
	}
	return nil
}

// AddVersion inserts a new version for an object
func AddVersion(db *sql.DB, bucketID, objectID, versionID string, metadata VersionMetadata) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	query := `INSERT INTO versions (version_id, object_id, bucket_id, metadata) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(query, versionID, objectID, bucketID, metadataJSON)
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
