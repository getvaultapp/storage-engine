package proofofinclusion

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/cbergoon/merkletree"
)

// Content represents the content stored in the Merkle tree
type Content struct {
	X string
}

// CalculateHash hashes the values of a Content
func (c Content) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(c.X)); err != nil {
		return nil, fmt.Errorf("failed to hash content: %w", err)
	}
	return h.Sum(nil), nil
}

// Equals tests for equality of two Contents
func (c Content) Equals(other merkletree.Content) (bool, error) {
	return c.X == other.(Content).X, nil
}

// BuildMerkleTree builds a Merkle tree from the given shards
func BuildMerkleTree(shards [][]byte) (*merkletree.MerkleTree, error) {
	var list []merkletree.Content
	for _, shard := range shards {
		list = append(list, Content{X: hex.EncodeToString(shard)})
	}

	tree, err := merkletree.NewTree(list)
	if err != nil {
		return nil, fmt.Errorf("failed to create Merkle tree: %w", err)
	}

	return tree, nil
}

// GetProof generates a proof of inclusion for a given shard
func GetProof(tree *merkletree.MerkleTree, shard []byte) (string, error) {
	proof, _, err := tree.GetMerklePath(Content{X: hex.EncodeToString(shard)})
	if err != nil {
		return "", fmt.Errorf("failed to get Merkle proof: %w", err)
	}

	return hex.EncodeToString(proof[len(proof)-1]), nil // Corrected to retrieve the last element in the proof path
}
