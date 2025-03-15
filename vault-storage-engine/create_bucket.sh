#!/bin/sh

# Replace these values with the actual bucket ID and owner
BUCKET_ID="bucket1"
OWNER="user1"

# URL of the server
SERVER_URL="http://localhost:8080"

# Read the token from the file
TOKEN=$(cat token.txt)

# Create the bucket
curl -X POST "$SERVER_URL/buckets" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "ID": "'"$BUCKET_ID"'",
    "Owner": "'"$OWNER"'"
  }'
