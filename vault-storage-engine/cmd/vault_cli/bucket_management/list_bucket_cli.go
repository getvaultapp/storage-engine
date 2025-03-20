package bucket_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvaultapp/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/vault-storage-engine/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func ListBucketCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	_, err := bucket.ListAllBuckets(db)

	if err != nil {
		return fmt.Errorf("failed to list buckets, %w", err)
	}
	return nil
}
