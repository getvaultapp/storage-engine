#!/bin/bash

# Create the certificates directory if it doesn't exist
mkdir -p vault-storage-engine/nodes/certs

# Generate the CA certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout vault-storage-engine/nodes/certs/ca.key -out vault-storage-engine/nodes/certs/ca.crt -subj "/C=US/ST=State/L=City/O=Organization/OU=Department/CN=example.com"

# Create a server private key
openssl genrsa -out vault-storage-engine/nodes/certs/server.key 2048

# Create a server certificate signing request (CSR)
openssl req -new -key vault-storage-engine/nodes/certs/server.key -out vault-storage-engine/nodes/certs/server.csr -subj "/C=US/ST=State/L=City/O=Organization/OU=Department/CN=example.com"

# Sign the server certificate with the CA certificate
openssl x509 -req -days 365 -in vault-storage-engine/nodes/certs/server.csr -CA vault-storage-engine/nodes/certs/ca.crt -CAkey vault-storage-engine/nodes/certs/ca.key -CAcreateserial -out vault-storage-engine/nodes/certs/server.crt

echo "Certificates generated and stored in vault-storage-engine/nodes/certs directory."
