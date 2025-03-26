package bucket

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

func ReadMetadataJson(db *sql.DB, bucketID, objectID, versionID string, filename string) error {
	//db, err := sql.Open("sqlite3", "./metadata.db")
	/* if err != nil {
		return fmt.Errorf("failed to open metadata database, %w", err)
	} */
	defer db.Close()

	query := `SELECT metadata FROM versions WHERE bucket_id = ? AND object_id = ? AND version_id = ?`
	rows, err := db.Query(query, bucketID, objectID, versionID)
	if err != nil {
		return fmt.Errorf("failed to get versionID and objectID, %w", err)
	}

	for rows.Next() {
		var metadataJSON string
		err = rows.Scan(&metadataJSON)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("no metadata found for object %s version (%s) in %s", objectID, versionID, bucketID)
			}
			return fmt.Errorf("failed to read metadata file: %w", err)
		}

		var metadata VersionMetadata
		err = json.Unmarshal([]byte(metadataJSON), &metadata)
		if err != nil {
			return fmt.Errorf("failed to unmarshal metadata file: %w", err)
		}

		outputJSON, err := json.MarshalIndent(metadata, "", " ")
		if err != nil {
			return fmt.Errorf("failed to read metadata: %w", err)
		}

		err = os.WriteFile(filename, outputJSON, 0644)
		if err != nil {
			return fmt.Errorf("failed to write json to file, %w", err)
		}

	}
	return nil
}
