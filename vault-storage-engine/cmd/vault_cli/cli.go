package vault_cli

import (
	"log"
	"os"

	"github.com/getvaultapp/vault-storage-engine/pkg/bucket"
	"github.com/getvaultapp/vault-storage-engine/pkg/config"
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
					return newBucketCommand(c, db)
				},
			},
			{
				Name:  "store-object",
				Usage: "Store objects in valid buckets. Usage: store-object <bucket_id> <file_path>",
				Action: func(c *cli.Context) error {
					return storeCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "get-object",
				Usage: "Retrieves a valid object from it's bucket. Usage: get-object <bucket_id> <object_id> <version_id>",
				Action: func(c *cli.Context) error {
					return retrieveCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "read-metadata-json",
				Usage: "Returns metadata.json for objects",
				Action: func(c *cli.Context) error {
					return bucket.ReadMetadataJson("metadata.json")
				},
			},
			{
				Name:  "list-buckets",
				Usage: "Lists all active buckets",
				Action: func(c *cli.Context) error {
					return listBucketCommand(c, db, cfg, logger)
				},
			},
			{
				Name:  "delete-object",
				Usage: "Deletes an object. Usage: delete-object <bucket_id> <object_id> <version_id>",
				Action: func(c *cli.Context) error {
					return deleteObject(c, db, cfg, logger)
				},
			},
			{
				Name:  "delete-bucket",
				Usage: "Deletes an entire bucket including all it's objects and their respective versions. Usage: delete-bucket <bucket-id>",
				Action: func(c *cli.Context) error {
					return deleteBucket(c, db, cfg, logger)
				},
			},
		},
	}

	/* r := api.SetupRouter(db) */

	/* go func() {
		if err := r.Run(cfg.ServerAddress); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()
	*/
	if err := app.Run(os.Args); err != nil {
		logger.Fatal("CLI failed", zap.Error(err))
	}
}
