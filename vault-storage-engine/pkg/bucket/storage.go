package bucket

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

const dbFile = "vault_metadata.db"

// InitDB initializes the SQLite database
func InitDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create tables
	schema := `
	CREATE TABLE IF NOT EXISTS buckets (
	    bucket_id TEXT PRIMARY KEY,
	    owner TEXT NOT NULL,
	    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS objects (
	    object_id TEXT PRIMARY KEY,
	    bucket_id TEXT NOT NULL,
	    latest_version TEXT,
	    FOREIGN KEY (bucket_id) REFERENCES buckets(bucket_id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS versions (
	    version_id TEXT NOT NULL,
	    object_id TEXT NOT NULL,
	    bucket_id TEXT NOT NULL,
	    metadata JSON NOT NULL,
	    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	    PRIMARY KEY (object_id, version_id),
	    FOREIGN KEY (bucket_id) REFERENCES buckets(bucket_id) ON DELETE CASCADE
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("Database initialized successfully.")
	return db, nil
}
