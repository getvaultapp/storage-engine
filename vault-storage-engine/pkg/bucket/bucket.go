/* package bucket

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
func CreateBucket(db *sql.DB, bucketID, owner string) error {
	query := `INSERT INTO buckets (bucket_id, owner) VALUES (?, ?)`
	_, err := db.Exec(query, bucketID, owner)
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
} */

package bucket

import (
	"database/sql"
	"errors"
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
		fmt.Printf("%s exists\n", bucketID)
		return errors.New("bucket exists")
	}

	query = `INSERT INTO buckets (bucket_id, owner) VALUES (?, "")`
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
