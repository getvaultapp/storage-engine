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

// ObjectType represents a miniature singleton of an object
type ObjectType struct {
	ObjectID      string
	Filename      string
	LatestVersion string
}

// ListObjects lists all objects in a bucket
func ListObjects(db *sql.DB, bucketID string) ([]ObjectType, error) {
	query := "SELECT id, filename, latest_version FROM objects WHERE bucket_id = $1"
	rows, err := db.Query(query, bucketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var objects []ObjectType
	for rows.Next() {
		var object ObjectType
		if err := rows.Scan(&object.ObjectID, &object.Filename, &object.LatestVersion); err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}

	return objects, nil
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
		latest_version_id := GetLatestVersion(db, objectID)
		query := "UPDATE objects SET latest_version = ? WHERE id = ? AND bucket_id = ? AND filename = ?"
		_, err = db.Exec(query, latest_version_id, objectID, bucketID, filename)
		if err != nil {
			return fmt.Errorf("failed to update object version, %s", err)
		}
		return nil
	}

	latest_version_id := GetLatestVersion(db, objectID)

	query = "INSERT INTO objects (id, bucket_id, filename, latest_version) VALUES (?, ?, ?, ?)"
	_, err = db.Exec(query, objectID, bucketID, filename, latest_version_id)
	if err != nil {
		return fmt.Errorf("failed to add object: %w", err)
	}

	return nil
}

func AddVersion(db *sql.DB, bucketID, objectID, versionID, rootVersion string, metadata VersionMetadata, data []byte) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to encode metadata: %w", err)
	}

	// Convert shard_locations map to JSON
	shardLocBytes, err := json.Marshal(metadata.ShardLocations)
	if err != nil {
		return fmt.Errorf("failed to encode shard locations: %w", err)
	}

	query := `
		INSERT INTO versions (
			version_id, object_id, bucket_id, root_version,
			metadata, data, shard_locations, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	_, err = db.Exec(query, versionID, objectID, bucketID, rootVersion, metadataJSON, data, string(shardLocBytes))
	if err != nil {
		return fmt.Errorf("failed to add version: %w", err)
	}

	// Update latest version
	_, err = db.Exec(`UPDATE objects SET latest_version = ? WHERE id = ?`, versionID, objectID)
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

func GetObjectMetadataAllVersions(db *sql.DB, objectID string) (map[string]VersionMetadata, error) {
	query := `SELECT version_id, metadata FROM versions WHERE object_id = ?`
	rows, err := db.Query(query, objectID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve metadata, %w", err)
	}
	defer rows.Close()

	metadata := make(map[string]VersionMetadata)
	for rows.Next() {
		var versionID string
		var metadataJSON string
		err := rows.Scan(&versionID, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metadata %w", err)
		}

		var objectMetadata VersionMetadata
		err = json.Unmarshal([]byte(metadataJSON), &objectMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata, %w", err)
		}
		metadata[versionID] = objectMetadata
	}

	return metadata, nil
}

// Pls review this, I believe we can still optimize it better
func GetLatestVersion(db *sql.DB, objectID string) string {
	query := `SELECT version_id FROM versions WHERE object_id = ? ORDER BY version_id DESC LIMIT 1`
	row := db.QueryRow(query, objectID)
	var latestVersionID string
	err := row.Scan(&latestVersionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ""
		}
		return ""
	}

	return latestVersionID
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

// Remove object from database
func DeleteObject(db *sql.DB, bucketID, objectID string) error {
	// Remove the object versions
	query := "DELETE FROM versions WHERE object_id = ?"
	_, err := db.Exec(query, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete objects: %w", err)
	}

	// Remove the objects
	query = "DELETE FROM objects WHERE id = ?"
	_, err = db.Exec(query, objectID)
	if err != nil {
		return fmt.Errorf("failed to delete the object, %w", err)
	}
	return nil
}

func DeleteObjectByVersion(db *sql.DB, bucketID, objectID, versionID string) error {
	query := "DELETE FROM versions WHERE object_id = ? AND version_id = ?"
	_, err := db.Exec(query, objectID, versionID)
	if err != nil {
		return fmt.Errorf("failed to delete object version, %w", err)
	}

	latest_version_id := GetLatestVersion(db, objectID)

	query = "UPDATE objects SET latest_version = ? WHERE id = ? AND bucket_id = ?"
	_, err = db.Exec(query, latest_version_id, objectID, bucketID)
	if err != nil {
		return fmt.Errorf("failed to update object version, %s", err)
	}

	return nil
}

func UpdateFileVersionIfItExists(db *sql.DB, originalFile, bucketID, objectID string) (string, string, error) {
	// This returns the filename of an object, to check if it matches with the originalFile (file argument)
	version := uuid.New()
	query := `SELECT filename FROM objects WHERE id = ? AND bucket_id = ?`
	rows, err := db.Query(query, objectID, bucketID)
	if err != nil {
		return "", "", fmt.Errorf("operation failed %w", err)
	}

	var filename string

	for rows.Next() {
		var foundFilename string
		err := rows.Scan(&foundFilename)
		if err != nil {
			return "", "", fmt.Errorf("no filename for object %s", objectID)
		}
		//fmt.Println(foundFilename)
		filename = foundFilename
	}
	if filename == originalFile {
		// Update the latest version in the metadata.db
		query := "UPDATE objects SET latest_version = ? WHERE id = ? AND bucket_id = ? AND filename = ?"
		_, err = db.Exec(query, version, objectID, bucketID, filename)
		if err != nil {
			return "", "", fmt.Errorf("failed to update object version, %s", err)
		}

	}

	return filename, version.String(), nil
}
