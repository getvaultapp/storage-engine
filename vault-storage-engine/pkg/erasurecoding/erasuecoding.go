/* package erasurecoding

import (
	"fmt"

	"github.com/klauspost/reedsolomon"
)

const (
	DataShards   = 8
	ParityShards = 6
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
} */

package erasurecoding

import (
	"bytes"

	"github.com/klauspost/reedsolomon"
)

var (
	DataShards   = 4
	ParityShards = 2
)

// Encode splits and encodes the data into shards.
func Encode(data []byte) ([][]byte, error) {
	enc, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, err
	}
	shards, err := enc.Split(data)
	if err != nil {
		return nil, err
	}
	if err = enc.Encode(shards); err != nil {
		return nil, err
	}
	return shards, nil
}

// Decode reconstructs the original data from shards.
func Decode(shards [][]byte) ([]byte, error) {
	enc, err := reedsolomon.New(DataShards, ParityShards)
	if err != nil {
		return nil, err
	}
	if err = enc.Reconstruct(shards); err != nil {
		return nil, err
	}
	// Join shards back into a single byte slice.
	var buf bytes.Buffer
	if err = enc.Join(&buf, shards, len(shards[0])*DataShards); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
