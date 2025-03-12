package bucket

import (
	"fmt"

	"github.com/getvault-mvp/vault-base/pkg/config"
)

// GetEncryptionKey retrieves the encryption key from the configuration
func GetEncryptionKey(cfg *config.Config) ([]byte, error) {
	key := cfg.EncryptionKey
	if key == "" {
		return nil, fmt.Errorf("encryption key not found in configuration")
	}
	return []byte(key), nil
}
