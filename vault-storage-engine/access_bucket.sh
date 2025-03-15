#!/bin/sh

# Replace this value with the actual bucket ID
BUCKET_ID="bucket1"

# URL of the server
SERVER_URL="http://localhost:8080"

# Read the token from the file
TOKEN=$(cat token.txt)

# Access the bucket with proper authorization
curl -X GET "$SERVER_URL/buckets/$BUCKET_ID" \
  -H "Authorization: Bearer $TOKEN"
