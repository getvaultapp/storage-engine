package bucket

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

const dbFile = "vault_metadata.db"

// InitDB initializes the SQLite database
// InitDB initializes the database if it doesn't exist and returns a connection to it.
func InitDB() (*sql.DB, error) {
	dbPath := "vault_metadata.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Database file does not exist, create and initialize it
		file, err := os.Create(dbPath)
		if err != nil {
			return nil, err
		}
		file.Close()
		log.Println("Database file created successfully.")
	} else {
		log.Println("Database file already exists.")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize database schema if necessary
	if err := initializeSchema(db); err != nil {
		return nil, err
	}

	log.Println("Database initialized successfully.")
	return db, nil
}

// initializeSchema sets up the database schema if it doesn't exist.
func initializeSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS buckets (
		id TEXT PRIMARY KEY,
		owner TEXT NOT NULL
	);
	CREATE TABLE IF NOT EXISTS objects (
		id TEXT PRIMARY KEY,
		bucket_id TEXT NOT NULL,
		data BLOB NOT NULL,
		FOREIGN KEY (bucket_id) REFERENCES buckets(id)
	);
	CREATE TABLE IF NOT EXISTS versions (
		id TEXT PRIMARY KEY,
		object_id TEXT NOT NULL,
		data BLOB NOT NULL,
		FOREIGN KEY (object_id) REFERENCES objects(id)
	);
	CREATE TABLE IF NOT EXISTS acl (
		resource_id TEXT,
		resource_type TEXT,
		user_id TEXT,
		permission TEXT,
		PRIMARY KEY (resource_id, resource_type, user_id)
	);
	`
	_, err := db.Exec(schema)
	return err
}
