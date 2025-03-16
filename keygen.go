package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
)

func KeyGen() {
	// Define the key size flag
	var keySize int
	flag.IntVar(&keySize, "size", 32, "Size of the encryption key in bytes (must be 16, 24, or 32)")

	// Parse the flags
	flag.Parse()

	// Validate the key size
	if keySize != 16 && keySize != 24 && keySize != 32 {
		log.Fatalf("Invalid key size: %d. Key size must be 16, 24, or 32 bytes.", keySize)
	}

	// Generate the encryption key
	key := make([]byte, keySize)
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("Failed to generate encryption key: %v", err)
	}

	// Print the encryption key in hexadecimal format
	fmt.Printf("Generated %d-byte encryption key: %s\n", keySize, hex.EncodeToString(key))
}
