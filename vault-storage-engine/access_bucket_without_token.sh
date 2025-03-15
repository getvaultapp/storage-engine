#!/bin/sh

# Replace this value with the actual bucket ID
BUCKET_ID="bucket1"

# URL of the server
SERVER_URL="http://localhost:8080"

# Access the bucket without proper authorization
curl -X GET "$SERVER_URL/buckets/$BUCKET_ID"
