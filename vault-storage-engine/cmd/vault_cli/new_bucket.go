package vault_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func newBucketCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() != 2 {
		return fmt.Errorf("usage: retrieve <bucket_id> <owner_id>")
	}

	bucketID := c.Args().Get(0)
	ownerID := c.Args().Get(1)

	err := bucket.CreateBucket(db, bucketID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to create new bucket, %w", err)
	}
	fmt.Printf("Succcessfully create bucket: \"%s\" for \"%s\"\n", bucketID, ownerID)

	return nil
}
