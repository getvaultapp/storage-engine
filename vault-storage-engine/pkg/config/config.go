package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// Config holds the configuration settings
type Config struct {
	ServerAddress      string `yaml:"server_address"`
	ShardStoreBasePath string `yaml:"shard_store_base_path"`
	EncryptionKey      string `yaml:"encryption_key"`
	Database           string `yaml:"database"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig() *Config {
	f, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("failed to open config file: %v", err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("failed to decode config file: %v", err)
	}

	// Check if the EncryptionKey is not empty
	if cfg.EncryptionKey == "" {
		log.Fatal("encryption key not found in configuration")
	}

	return &cfg
}
