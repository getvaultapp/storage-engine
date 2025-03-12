package erasurecoding

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
)

const (
	DataShards   = 4
	ParityShards = 2
)

// Encode encodes data using Reed-Solomon erasure coding
func Encode(data []byte) ([][]byte, error) {
	enc, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, fmt.Errorf("failed to create encoder: %w", err)
	}

	shards, err := enc.Split(data)
	if err != nil {
		return nil, fmt.Errorf("failed to split data into shards: %w", err)
	}

	err = enc.Encode(shards)
	if err != nil {
		return nil, fmt.Errorf("failed to encode data: %w", err)
	}

	return shards, nil
}

// Decode decodes data using Reed-Solomon erasure coding
func Decode(shards [][]byte) ([]byte, error) {
	enc, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, fmt.Errorf("failed to create decoder: %w", err)
	}

	err = enc.Reconstruct(shards)
	if err != nil {
		return nil, fmt.Errorf("failed to reconstruct data: %w", err)
	}

	data, err := enc.Join(nil, shards, len(shards[0])*DataShards)
	if err != nil {
		return nil, fmt.Errorf("failed to join shards: %w", err)
	}

	return data, nil
}
