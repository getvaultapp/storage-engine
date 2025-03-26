package bucket_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func ListObjectCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	bucketID := c.Args().Get(0)
	objects, err := bucket.ListObjects(db, bucketID)
	for _, object := range objects {
		fmt.Println(object)
	}

	if err != nil {
		return fmt.Errorf("failed to list buckets, %w", err)
	}
	return nil
}
