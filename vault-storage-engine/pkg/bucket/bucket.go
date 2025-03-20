package bucket

import (
	"database/sql"
	"fmt"
	"time"
)

// Bucket represents a storage bucket
type Bucket struct {
	ID        string
	Owner     string
	CreatedAt time.Time
}

// CreateBucket inserts a new bucket into the database
func CreateBucket(db *sql.DB, bucketID string, owner string) error {
	var bucketExists bool

	// Check if the Bucket exists
	query := "SELECT EXISTS(SELECT 1 FROM buckets WHERE bucket_id = ?)"
	err := db.QueryRow(query, bucketID).Scan(&bucketExists)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists, %w", err)
	}

	if bucketExists {
		return nil
	}

	query = `INSERT INTO buckets (bucket_id, owner) VALUES (?, ?)`
	_, err = db.Exec(query, bucketID, owner)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// GetBucket retrieves a bucket by ID
func GetBucket(db *sql.DB, bucketID string) (*Bucket, error) {
	query := `SELECT bucket_id, owner, created_at FROM buckets WHERE bucket_id = ?`
	row := db.QueryRow(query, bucketID)

	var bucket Bucket
	err := row.Scan(&bucket.ID, &bucket.Owner, &bucket.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("bucket not found")
		}
		return nil, fmt.Errorf("failed to get bucket: %w", err)
	}

	return &bucket, nil
}

func ListAllBuckets(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT bucket_id FROM buckets")
	if err != nil {
		return nil, fmt.Errorf("error reading row, %w", err)
	}
	defer rows.Close()

	var bucketIDs []string
	for rows.Next() {
		var bucket_id string
		err := rows.Scan(&bucket_id)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve bucket ids")
		}

		bucketIDs = append(bucketIDs, bucket_id)
	}
	for _, bucket_id := range bucketIDs {
		fmt.Println("* ", bucket_id)
	}

	return bucketIDs, nil
}

// Returns all the objects in a bucket
func GetObjectsInBucket(db *sql.DB, bucketID string) ([]string, error) {
	query := "SELECT id FROM objects WHERE bucket_id = ?"
	rows, err := db.Query(query, bucketID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve objects in bucket, %w", err)
	}
	defer rows.Close()

	var objectIDs []string
	for rows.Next() {
		var objectID string
		err := rows.Scan(&objectID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan object ID: %w", err)
		}
		objectIDs = append(objectIDs, objectID)
	}
	return objectIDs, nil
}

// Deletes all the contents of a bucket
func DeleteBucket(db *sql.DB, bucketID string) error {
	objects, err := GetObjectsInBucket(db, bucketID)
	if err != nil {
		return fmt.Errorf("failed to get objects in bucket, %w", err)
	}
	for _, objectID := range objects {
		err := DeleteObject(db, bucketID, objectID)
		if err != nil {
			return fmt.Errorf("failed to delete object, %w", err)
		}
	}

	// Remove bucket from database
	query := "DELETE FROM buckets WHERE bucket_id = ?"
	_, err = db.Exec(query, bucketID)
	if err != nil {
		return fmt.Errorf("failed to delete bucket, %w", err)
	}
	return nil
}
