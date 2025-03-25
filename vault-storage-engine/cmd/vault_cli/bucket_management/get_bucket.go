package bucket_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/urfave/cli/v2"
)

func GetBucketCommand(c *cli.Context, db *sql.DB) error {
	if c.NArg() != 2 {
		return fmt.Errorf("usage: get-bucket <bucket_id> <owner>")
	}

	bucketID := c.Args().Get(0)

	bucket, err := bucket.GetBucket(db, bucketID)
	if err != nil {
		return fmt.Errorf("failed to get bucket, %w", err)
	}
	println(bucket)

	return nil
}
