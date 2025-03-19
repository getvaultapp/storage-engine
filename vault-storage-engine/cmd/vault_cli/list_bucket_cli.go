package vault_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func listBucketCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	_, err := bucket.ListAllBuckets(db)

	if err != nil {
		return fmt.Errorf("failed to list buckets, %w", err)
	}
	return nil
}
