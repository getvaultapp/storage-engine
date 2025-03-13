package config

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"

	"github.com/spf13/viper"
	//"gopkg.in/yaml.v2"
)

// Config holds the configuration settings
type Config struct {
	ServerAddress      string `yaml:"server_address"`
	ShardStoreBasePath string `yaml:"shard_store_base_path"`
	EncryptionKey      string `yaml:"encryption_key"`
	Database           string `yaml:"database"`
}

// LoadConfig loads the configuration from a YAML file
func LoadConfig() (*Config, error) {
	/* f, err := os.Open("config.yaml")
	if err != nil {
		log.Fatalf("failed to open config file: %v", err)
	}
	defer f.Close()
	*/

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	/* decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		log.Fatalf("failed to decode config file: %v", err)
	} */

	return &config, nil
}

// InitializeCipher checks the validity of en encryption key
func InitializeCipher(key string) (cipher.AEAD, error) {
	decodedKey, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key: %v", err)
	}

	block, err := aes.NewCipher(decodedKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	return gcm, nil
}
