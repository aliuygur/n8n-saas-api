#!/bin/bash

# Test deployment script for n8n
# This script replaces placeholders with test values and deploys to Kubernetes

# Generate random suffix for non-idempotent deployments
RANDOM_SUFFIX=$(head /dev/urandom | LC_ALL=C tr -dc a-z0-9 | head -c 6)

# Test values
NAMESPACE="n8n-test-${RANDOM_SUFFIX}"
ENCRYPTION_KEY="test-encryption-key-12345678901234567890"
BASE_URL="https://test.n8n.example.com"

# Read the template and replace placeholders
cat internal/provisioning/n8ntemplates/templates/n8n.yaml | \
  sed "s/PLACEHOLDER_NAMESPACE/${NAMESPACE}/g" | \
  sed "s/PLACEHOLDER_ENCRYPTION_KEY/${ENCRYPTION_KEY}/g" | \
  sed "s|PLACEHOLDER_BASE_URL|${BASE_URL}|g" | \
  kubectl apply -f -

echo "Deployed n8n to namespace: ${NAMESPACE}"
echo "Base URL: ${BASE_URL}"
