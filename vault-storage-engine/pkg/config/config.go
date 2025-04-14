package config

import (
	"encoding/hex"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

// Config holds the configuration settings
type Config struct {
	ServerAddress      string   `yaml:"server_address"`
	ShardStoreBasePath string   `yaml:"shard_store_base_path"`
	EncryptionKey      []byte   `yaml:"-"`
	EncryptionKeyHex   string   `yaml:"encryption_key"`
	Database           string   `yaml:"db"`
	ShardLocations     []string `yaml:"shardLocations"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig() *Config {
	f, err := os.Open("/home/tnxl/storage-engine/vault-storage-engine/config.yaml")
	if err != nil {
		log.Fatalf("failed to open config file: %v", err)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("failed to decode config file: %v", err)
	}

	// Decode the hex-encoded encryption key
	key, err := hex.DecodeString(cfg.EncryptionKeyHex)
	if err != nil {
		log.Fatalf("failed to decode encryption key: %v", err)
	}

	// Ensure the key length is valid for AES (16, 24, or 32 bytes)
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		log.Fatalf("invalid encryption key size: %d bytes", len(key))
	}

	cfg.EncryptionKey = key

	return &cfg
}
