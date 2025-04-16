package bucket

import (
	"database/sql"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// InitDB initializes the SQLite database
// InitDB initializes the database if it doesn't exist and returns a connection to it.
func InitDB() (*sql.DB, error) {
	dbPath := "metadata.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Database file does not exist, create and initialize it
		file, err := os.Create(dbPath)
		if err != nil {
			return nil, err
		}
		file.Close()
		//log.Println("Database file created successfully.")
	} else {
		//log.Println("Database file already exists.")
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Initialize database schema if necessary
	if err := initializeSchema(db); err != nil {
		return nil, err
	}

	//log.Println("Database initialized successfully.")
	return db, nil
}

// initializeSchema sets up the database schema if it doesn't exist.
func initializeSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
    	id INTEGER PRIMARY KEY AUTOINCREMENT,
    	username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
    	password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS buckets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		bucket_id TEXT NOT NULL,
		owner TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE IF NOT EXISTS objects (
		id TEXT PRIMARY KEY,
		filename TEXT NOT NULL,
		bucket_id TEXT NOT NULL,
		latest_version TEXT,
		FOREIGN KEY (bucket_id) REFERENCES buckets(id)
	);
	CREATE TABLE IF NOT EXISTS versions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		object_id TEXT NOT NULL,
		version_id TEXT NOT NULL,
		bucket_id TEXT NOT NULL,
		metadata TEXT NOT NULL,
		root_version TEXT NOT NULL,
		data BLOB NOT NULL,
		shard_locations BLOB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
