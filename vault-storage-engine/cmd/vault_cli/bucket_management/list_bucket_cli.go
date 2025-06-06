package bucket_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func ListBucketCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() != 1 {
		return fmt.Errorf("usage: list-bucket <owne_id>")
	}

	owner := c.Args().Get(0)

	_, err := bucket.ListAllBuckets(db, owner)

	if err != nil {
		return fmt.Errorf("failed to list buckets, %w", err)
	}
	return nil
}
