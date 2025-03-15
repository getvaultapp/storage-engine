package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/getvault-mvp/vault-base/cmd/vault_cli"
	"github.com/getvault-mvp/vault-base/pkg/api"
	"github.com/getvault-mvp/vault-base/pkg/bucket"
	"github.com/getvault-mvp/vault-base/pkg/config"
	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

func main() {
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
				Name:    "store",
				Aliases: []string{"s"},
				Usage:   "Store data. Usage: store <bucket_id> <file_path>",
				Action: func(c *cli.Context) error {
					return vault_cli.StoreCommand(c, db, cfg, logger)
				},
			},
			{
				Name:    "retrieve",
				Aliases: []string{"r"},
				Usage:   "Retrieve data. Usage: retrieve <bucket_id> <object_id> <version_id>",
				Action: func(c *cli.Context) error {
					return vault_cli.RetrieveCommand(c, db, cfg, logger)
				},
			},
		},
	}

	// Start the server in a goroutine
	go func() {
		r := api.SetupRouter(db)
		if err := r.Run(cfg.ServerAddress); err != nil {
			log.Fatalf("Failed to run server: %v", err)
		}
	}()

	// Run the CLI app
	go func() {
		if err := app.Run(os.Args); err != nil {
			logger.Fatal("CLI failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("Shutting down server...")
}
