package utils

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/getvaultapp/storage-engine/vault-storage-engine/pkg/config"
	"github.com/spf13/viper"
)

// Helper function to convert a slice to a map
func ConvertSliceToMap(slice []string) map[string]string {
	result := make(map[string]string)
	for i, v := range slice {
		key := fmt.Sprintf("key_%d", i) // Use a suitable key generation logic
		result[key] = v
	}
	return result
}

// ConvertViperToConfig converts a viper.Viper instance to a config.Config instance
func ConvertViperToConfig(v *viper.Viper) *config.Config {
	cfg := &config.Config{
		ServerAddress:      v.GetString("server_address"),
		ShardStoreBasePath: v.GetString("shard_store_base_path"),
		EncryptionKeyHex:   v.GetString("encryption_key"),
		Database:           v.GetString("database"),
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

	return cfg
}
