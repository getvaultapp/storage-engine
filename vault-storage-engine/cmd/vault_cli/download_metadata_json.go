package vault_cli

import (
	"database/sql"
	"fmt"

	"github.com/getvaultapp/vault-storage-engine/pkg/bucket"
	"github.com/urfave/cli/v2"
)

func readMetadataJsonCommand(c *cli.Context, db *sql.DB) error {
	if c.NArg() != 3 {
		return fmt.Errorf("usage: read-metadata-json <bucket_id> <object_id> <version_id>")
	}

	bucketID := c.Args().Get(0)
	objectID := c.Args().Get(1)
	version := c.Args().Get(2)

	filename := objectID + "-" + version + "-metadata.json"

	err := bucket.ReadMetadataJson(bucketID, objectID, version, filename)
	if err != nil {
		return fmt.Errorf("failed to create new bucket, %w", err)
	}
	fmt.Printf("Metadata stored in : \"%s\"\n", filename)

	return nil
}
