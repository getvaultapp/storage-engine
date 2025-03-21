package bucket_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvaultapp/vault-storage-engine/pkg/bucket"
	"github.com/urfave/cli/v2"
)

func NewBucketCommand(c *cli.Context, db *sql.DB) error {
	if c.NArg() != 2 {
		return fmt.Errorf("usage: create-bucket <bucket_id> <owner_id>")
	}

	bucketID := c.Args().Get(0)
	ownerID := c.Args().Get(1)

	err := bucket.CreateBucket(db, bucketID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to create new bucket, %w", err)
	}
	fmt.Printf("Succcessfully created bucket: \"%s\" for \"%s\"\n", bucketID, ownerID)

	return nil
}
