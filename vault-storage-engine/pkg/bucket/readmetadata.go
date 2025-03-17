package bucket

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
)

func ReadMetadataJson(filename string) error {
	db, err := sql.Open("sqlite3", "./metadata.db")
	if err != nil {
		return fmt.Errorf("failed to open metadata database, %w", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT metadata FROM versions")
	if err != nil {
		return fmt.Errorf("failed to read table versions, %w", err)
	}

	var allMetadata []VersionMetadata

	for rows.Next() {
		var metadataJSON string
		err := rows.Scan(&metadataJSON)
		if err != nil {
			return fmt.Errorf("operation failed, %w", err)
		}

		var metadata VersionMetadata
		err = json.Unmarshal([]byte(metadataJSON), &metadata)
		if err != nil {
			return fmt.Errorf("operation failed, %w", err)
		}

		allMetadata = append(allMetadata, metadata)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error during row iteration: %w", err)
	}

	outputJSON, err := json.MarshalIndent(allMetadata, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata to JSON: %w", err)
	}
	err = os.WriteFile(filename, outputJSON, 0644)
	if err != nil {
		return fmt.Errorf("failed to write json to file, %w", err)

	}
	return nil
}
