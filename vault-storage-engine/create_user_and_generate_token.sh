#!/bin/sh

# Replace these values with the actual username and password
USERNAME="user1"
PASSWORD="password" # This is the plain text password

# URL of the server
SERVER_URL="http://localhost:8080"

# Perform the login and get the JWT token
RESPONSE=$(curl -s -X POST "$SERVER_URL/login" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "'"$USERNAME"'",
    "password": "'"$PASSWORD"'"
  }')

# Extract the token from the response
TOKEN=$(echo $RESPONSE | jq -r '.token')

# Check if the token is null
if [ "$TOKEN" = "null" ]; then
  echo "Failed to retrieve JWT token. Response: $RESPONSE"
  exit 1
fi

# Print the token
echo "JWT Token: $TOKEN"

# Save the token to a file for later use
echo $TOKEN > token.txt
