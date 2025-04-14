#!/bin/bash

# Define paths to your compiled binaries
CONSTRUCTION_BIN=/home/tnxl/storage-engine/vault-storage-engine/simulation/bin/construction-node
STORAGE_BIN=/home/tnxl/storage-engine/vault-storage-engine/simulation/bin/storage-node
CERT_DIR=/home/tnxl/storage-engine/vault-storage-engine/certs

# Kill any existing nodes from previous runs
pkill -f construction-node
pkill -f storage-node

# Make logs and data directories
mkdir -p logs data/s1 data/s2

echo "[+] Starting Vault nodes (1 construction, 2 storage) with embedded discovery + P2P..."

### --- Node 1: Construction Node ---
NODE_ID=c1
NODE_TYPE=construction
CONSTRUCTION_PORT=8080
DISCOVERY_PORT=9000

NODE_ID=$NODE_ID NODE_TYPE=$NODE_TYPE CONSTRUCTION_PORT=$CONSTRUCTION_PORT DISCOVERY_PORT=$DISCOVERY_PORT \
  $CONSTRUCTION_BIN > logs/construct_c1.log 2>&1 &

### --- Node 2: Storage Node s1 ---
NODE_ID=s1
NODE_TYPE=storage
STORAGE_PORT=8081
STORAGE_DISCOVERY_PORT=9001
SHARD_STORE_BASE_PATH=./data/s1

NODE_ID=$NODE_ID NODE_TYPE=$NODE_TYPE STORAGE_PORT=$STORAGE_PORT STORAGE_DISCOVERY_PORT=$STORAGE_DISCOVERY_PORT \
  SHARD_STORE_BASE_PATH=$SHARD_STORE_BASE_PATH \
  $STORAGE_BIN > logs/storage_s1.log 2>&1 &

### --- Node 3: Storage Node s2 ---
NODE_ID=s2
NODE_TYPE=storage
STORAGE_PORT=8082
STORAGE_DISCOVERY_PORT=9002
SHARD_STORE_BASE_PATH=./data/s2

NODE_ID=$NODE_ID NODE_TYPE=$NODE_TYPE STORAGE_PORT=$STORAGE_PORT STORAGE_DISCOVERY_PORT=$STORAGE_DISCOVERY_PORT \
  SHARD_STORE_BASE_PATH=$SHARD_STORE_BASE_PATH \
  $STORAGE_BIN > logs/storage_s2.log 2>&1 &

sleep 5

echo "[+] Nodes started. Fetching peer and registry info from each node's discovery service..."

for port in 9000 9001 9002; do
  echo ""
  echo "🔎 Discovery on port :$port"
  echo "→ Registered Nodes:"
  curl --insecure -k --cert $CERT_DIR/server.crt \
       --key $CERT_DIR/server.key \
       --cacert $CERT_DIR/ca.crt \
       https://localhost:$port/nodes | jq

  echo "→ Gossip Peers:"
  curl --insecure -k --cert $CERT_DIR/server.crt \
       --key $CERT_DIR/server.key \
       --cacert $CERT_DIR/ca.crt \
       https://localhost:$port/gossip/peers | jq
done

echo ""
echo "[✓] Simulation complete. Logs stored in ./logs/"
