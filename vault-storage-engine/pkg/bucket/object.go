package bucket

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// Object represents a stored file
type Object struct {
	ID            string
	BucketID      string
	Filename      string
	LatestVersion string
}

// VersionMetadata represents the metadata for a version
type VersionMetadata struct {
	BucketID       string            `json:"bucket_id"`
	ObjectID       string            `json:"object_id"`
	VersionID      string            `json:"file_version"`
	Filename       string            `json:"filename"`
	Filesize       string            `json:"filesize"`
	Format         string            `json:"file_formart"`
	CreationDate   string            `json:"creation_date"`
	Data           []byte            `json:"data"`
	ShardLocations map[string]string `json:"shard_locations"`
	Proofs         map[string]string `json:"proofs"`
}

// AddObject adds an object to the database if it doesn't already exist
func AddObject(db *sql.DB, bucketID, objectID, filename string) error {
	var objectExists bool
	query := "SELECT EXISTS(SELECT 1 FROM objects WHERE id = ? AND bucket_id = ? AND filename = ?)"
	err := db.QueryRow(query, objectID, bucketID, filename).Scan(&objectExists)
	if err != nil {
		return fmt.Errorf("failed to check if object exists: %w", err)
	}

	if objectExists {
		//err := updateObjectVersion(db, objectID, bucketID)
		versionID := uuid.New()
		query := "UPDATE objects SET latest_version = ? WHERE id = ? AND bucket_id = ? AND filename = ?"
		_, err = db.Exec(query, versionID, objectID, bucketID, filename)
		if err != nil {
			return fmt.Errorf("failed to update object version, %s", err)
		}

		return nil
	}

	var filenameExists bool
	query = "SELECT EXISTS(SELECT filename FROM objects WHERE id = ? AND bucket_id = ? AND filename = ?)"
	err = db.QueryRow(query, objectID, bucketID, filename).Scan(&filenameExists)
	if err != nil {
		return fmt.Errorf("failed to object update version: %w", err)
	}

	if filenameExists {
		//err := updateObjectVersion(db, objectID, bucketID)
		versionID := uuid.New()
		query := "UPDATE objects SET latest_version = ? WHERE id = ? AND bucket_id = ? AND filename = ?"
		_, err = db.Exec(query, versionID, objectID, bucketID, filename)
		if err != nil {
			return fmt.Errorf("failed to update object version, %s", err)
		}

		return nil
	}

	query = "INSERT INTO objects (id, bucket_id, filename) VALUES (?, ?, ?)"
	_, err = db.Exec(query, objectID, bucketID, filename)
	if err != nil {
		return fmt.Errorf("failed to add object: %w", err)
	}

	return nil
}

// AddVersion inserts a new version for an object
func AddVersion(db *sql.DB, bucketID, objectID, versionID, rootVersion string, metadata VersionMetadata, data []byte) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	query := `INSERT INTO versions (version_id, object_id, bucket_id, root_version, metadata, data) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, versionID, objectID, bucketID, rootVersion, metadataJSON, data)
	if err != nil {
		return fmt.Errorf("failed to add version: %w", err)
	}

	_, err = db.Exec(`UPDATE objects SET latest_version = ? WHERE id = ?`, versionID, objectID)
	if err != nil {
		return fmt.Errorf("failed to update object latest version: %w", err)
	}

	// Update the latest version for the object
	updateQuery := `UPDATE objects SET latest_version = ? WHERE id = ?`
	_, err = db.Exec(updateQuery, versionID, objectID)
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

func GetRootVersion(db *sql.DB, objectID string) (string, error) {
	// Do nothing yet
	var rootVersion string
	query := `SELECT version_id FROM versions WHERE object_id = ? ORDER BY version_id ASC LIMIT 1`
	row := db.QueryRow(query, objectID)
	err := row.Scan(&rootVersion)
	if err != nil {
		// Handle error or set default root version
		rootVersion = "initial_version"
	}
	return rootVersion, nil
}

func DeleteObject(db *sql.DB, bucketID, objectID string) error {
	// Remove the object versions
	query := "DELETE FROM versions WHERE object_id = ?"
	_, err := db.Exec(query, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete object versions: %w", err)
	}

	// Remove the objects
	query = "DELETE FROM objects WHERE id = ?"
	_, err = db.Exec(query, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete the object, %w", err)
	}
	return nil
}
