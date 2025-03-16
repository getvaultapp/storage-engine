package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type VersionMetadata struct {
	Data           []byte            `json:"data"`
	ShardLocations map[string]string `json:"shard_locations"`
	Proofs         map[string]string `json:"proofs"`
}

func main() {
	// Open the SQLite database
	db, err := sql.Open("sqlite3", "./metadata.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Query the metadata column from the versions table
	rows, err := db.Query("SELECT metadata FROM versions")
	if err != nil {
		log.Fatal("Failed to query metadata:", err)
	}
	defer rows.Close()

	var allMetadata []VersionMetadata

	// Iterate over the rows and decode the JSON metadata
	for rows.Next() {
		var metadataJSON string
		err := rows.Scan(&metadataJSON)
		if err != nil {
			log.Fatal("Failed to scan metadata:", err)
		}

		var metadata VersionMetadata
		err = json.Unmarshal([]byte(metadataJSON), &metadata)
		if err != nil {
			log.Fatal("Failed to unmarshal metadata:", err)
		}

		allMetadata = append(allMetadata, metadata)
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		log.Fatal("Error during row iteration:", err)
	}

	// Convert all metadata to JSON
	outputJSON, err := json.MarshalIndent(allMetadata, "", "  ")
	if err != nil {
		log.Fatal("Failed to marshal metadata to JSON:", err)
	}

	// Write the JSON to a file
	err = os.WriteFile("metadata.json", outputJSON, 0644)
	if err != nil {
		log.Fatal("Failed to write JSON to file:", err)
	}

	fmt.Println("Metadata JSON has been written to metadata.json")
}
