package config

/* import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
) */

/* type Config struct {
	ServerAddress      string
	ShardStoreBasePath string
	EncryptionKey      string
	Database           string
} */

/* func loadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
*/

/* func initializeCipher(key string) (cipher.AEAD, error) {
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
} */
