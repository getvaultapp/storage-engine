package main

import (
	"github.com/getvault-mvp/vault-base/pkg/datastorage"
	"github.com/getvault-mvp/vault-base/pkg/proofofinclusion"
)

func VerifyShard(metadatafile string, store sharding.ShardStore, logger *zap.Logger) error {
	// Read proofs from metadata file
	preProofs, err := readProofs(metadatafile)
	if err != nil {
		return err
	}

	// Read metadata file
	dataID, err := MetadataFileReader(metadatafile, "dataID")
	if err != nil {
		return fmt.Errorf("error reading metadata file: %w", err)
	}

	// Get shard locations
	locations := make([]string, 14)
	for i := 0; i < 14; i++ {
		key := fmt.Sprintf("shard_%d", i)
		location, err := MetadataFileReader(metadatafile, key)
		if err != nil {
			return fmt.Errorf("error reading shard location from metadata file: %w", err)
		}
		locations[i] = location
	}

	// Retrieve shards
	shards := make([][]byte, len(locations))
	for i, location := range locations {
		shard, err := store.RetrieveShard(dataID, i, location)
		if err != nil {
			logger.Warn("Shard retrieval failed", zap.Int("index", i), zap.String("location", location), zap.Error(err))
			continue
		}
		shards[i] = shard
	}

	// Build Merkle tree
	tree, err := proofofinclusion.BuildMerkleTree(shards)
	if err != nil {
		return fmt.Errorf("failed to build Merkle tree: %w", err)
	}

	// Verify shards
	for i, shard := range shards {
		if shard == nil {
			continue
		}

		preProof, ok := preProofs[i]
		if !ok {
			return fmt.Errorf("missing proof for shard %d", i)
		}

		postProof, err := proofofinclusion.GetProof(tree, shard)
		if err != nil {
			return fmt.Errorf("failed to recompute proof for shard %d: %w", i, err)
		}

		if !bytes.Equal(preProof, []byte(postProof)) {
			logger.Warn("Shard verification failed", zap.Int("shard", i))
			return fmt.Errorf("shard verification failed for shard %d", i)
		}
	}

	return nil
}
