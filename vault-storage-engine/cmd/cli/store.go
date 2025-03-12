package main

/* import (
	"fmt"
	"os"
	"path/filepath"

	"database/sql"

	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/getvault-mvp/vault-base/pkg/sharding"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func storeCommand(c *cli.Context, db *sql.DB, cfg *config.Config, logger *zap.Logger) error {
	if c.NArg() < 2 {
		return fmt.Errorf("usage: store <bucket_id> <file_path>")
	}

	bucketID := c.Args().Get(0)
	filePath := c.Args().Get(1)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	store := sharding.NewLocalShardStore(cfg.ShardStoreBasePath)
	versionID, err := datastorage.StoreData(db, data, bucketID, filepath.Base(filePath), "uploaded_file", store, cfg, []string{}, logger)
	if err != nil {
		return fmt.Errorf("store failed: %w", err)
	}

	fmt.Printf("Stored file as version %s in bucket %s\n", versionID, bucketID)
	return nil
}
*/
