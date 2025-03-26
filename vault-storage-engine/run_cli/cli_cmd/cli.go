package vault_cli

import (
	"log"
	"os"

	bucket_cli "github.com/getvaultapp/storage-engine/vault-storage-engine/cmd/vault_cli/bucket_management"
	metadata_cli "github.com/getvaultapp/storage-engine/vault-storage-engine/cmd/vault_cli/handling_metadata"
	object_cli "github.com/getvaultapp/storage-engine/vault-storage-engine/cmd/vault_cli/object_management"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

// This function contains the full working CLI, from storage to retrieval and creating a new bucket instance
func RunCli() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg := config.LoadConfig()

	db, err := bucket.InitDB()
	if err != nil {
		log.Fatal(err)
	}

	app := &cli.App{
		Name:  "Vault",
		Usage: "Store and Retrieve Data With Vault Storage Engine",
		Commands: []*cli.Command{
			{
				Name:  "create-bucket",
				Usage: "Create an empty bucket. Usage: create-bucket <bukcet_id> <owner_id>",
				Action: func(c *cli.Context) error {
					return bucket_cli.NewBucketCommand(c, db)
				},
			},
			{
				Name:  "get-bucket",
				Usage: "Get a bucket by ID Usage: get-bucket <bucket-id>",
				Action: func(c *cli.Context) error {
					return bucket_cli.GetBucketCommand(c, db)
				},
			},
			{
				Name:  "store-object",
				Usage: "Store objects in valid buckets. Usage: store-object <bucket_id> <file_path>",
				Action: func(c *cli.Context) error {
					return object_cli.StoreCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "get-object",
				Usage: "Retrieves a valid object from it's bucket. Usage: get-object <bucket_id> <object_id> <version_id>",
				Action: func(c *cli.Context) error {
					return object_cli.RetrieveCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "update-object",
				Usage: "Updates the version of a object. Usage: update-object <bucket_id> <object_id> <filename>",
				Action: func(c *cli.Context) error {
					return object_cli.UpdateByVersion(c, db, cfg, logger)
				},
			},
			{
				Name:  "read-metadata-json",
				Usage: "Returns metadata.json for objects",
				Action: func(c *cli.Context) error {
					return metadata_cli.ReadMetadataJsonCommand(c, db)
				},
			},
			{
				Name:  "list-buckets",
				Usage: "Lists all active buckets",
				Action: func(c *cli.Context) error {
					return bucket_cli.ListBucketCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "delete-object",
				Usage: "Deletes an all versions of an object. Usage: delete-object <bucket_id> <object_id>",
				Action: func(c *cli.Context) error {
					return object_cli.DeleteObject(c, db, cfg, logger)
				},
			},
			{
				Name:  "delete-object-version",
				Usage: "Deletes a version of an object. Usage: delete-object-version <bucket_id> <object_id> <object_version_id>",
				Action: func(c *cli.Context) error {
					return object_cli.DeleteObjectByVersion(c, db, cfg, logger)
				},
			},
			{
				Name:  "delete-bucket",
				Usage: "Deletes an entire bucket including all it's objects and their respective versions. Usage: delete-bucket <bucket-id>",
				Action: func(c *cli.Context) error {
					return bucket_cli.DeleteBucket(c, db, cfg, logger)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Fatal("CLI failed", zap.Error(err))
	}
}
