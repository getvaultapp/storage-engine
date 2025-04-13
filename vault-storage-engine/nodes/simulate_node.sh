#!/bin/bash

# Define base directory
NODE_BINARY=./vault-node  # <- Replace with your actual compiled binary path

# Kill any previous runs
pkill -f vault-node

echo "[+] Starting 3 Vault nodes with embedded discovery and gossip..."

# Node 1 - Construction
NODE_ID=c1
NODE_TYPE=construction
API_PORT=8080
DISCOVERY_PORT=9000
CERTS_PATH=./certs

NODE_ID=$NODE_ID NODE_TYPE=$NODE_TYPE CONSTRUCTION_PORT=$API_PORT DISCOVERY_PORT=$DISCOVERY_PORT \
  $NODE_BINARY > logs/node1.log 2>&1 &

# Node 2 - Storage A
NODE_ID=s1
NODE_TYPE=storage
API_PORT=8081
DISCOVERY_PORT=9001

NODE_ID=$NODE_ID NODE_TYPE=$NODE_TYPE STORAGE_PORT=$API_PORT STORAGE_DISCOVERY_PORT=$DISCOVERY_PORT \
  SHARD_STORE_BASE_PATH=./data/s1 \
  $NODE_BINARY > logs/node2.log 2>&1 &

# Node 3 - Storage B
NODE_ID=s2
NODE_TYPE=storage
API_PORT=8082
DISCOVERY_PORT=9002

NODE_ID=$NODE_ID NODE_TYPE=$NODE_TYPE STORAGE_PORT=$API_PORT STORAGE_DISCOVERY_PORT=$DISCOVERY_PORT \
  SHARD_STORE_BASE_PATH=./data/s2 \
  $NODE_BINARY > logs/node3.log 2>&1 &

sleep 3

echo "[+] Nodes started! Checking node peer tables..."

for i in 9000 9001 9002; do
  echo ""
  echo ">>> Node on Discovery Port :$i"
  curl -s https://localhost:$i/nodes --insecure | jq
  curl -s https://localhost:$i/gossip/peers --insecure | jq
done

echo ""
echo "[✓] Simulation complete. Logs stored in ./logs/"
