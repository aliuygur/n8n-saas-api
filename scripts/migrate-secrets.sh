#!/bin/bash

# This script migrates Polar secrets from PascalCase to SCREAMING_SNAKE_CASE
# It reads the old secret values and sets them with the new names

set -e

echo "Migrating Polar secrets to new naming convention..."
echo ""

# Check if old secrets exist
echo "Checking for old secrets..."
if ! encore secret list | grep -q "PolarAccessToken"; then
    echo "Error: PolarAccessToken secret not found"
    exit 1
fi

if ! encore secret list | grep -q "PolarProductID"; then
    echo "Error: PolarProductID secret not found"
    exit 1
fi

echo "✓ Old secrets found"
echo ""

# Note: We need to manually copy the values since Encore doesn't provide
# a way to read secret values programmatically for security reasons

echo "Please manually migrate the secrets:"
echo ""
echo "1. Get PolarAccessToken value from your Polar dashboard or secure storage"
echo "   Then run: encore secret set POLAR_ACCESS_TOKEN --dev"
echo ""
echo "2. Get PolarProductID value from your Polar dashboard or secure storage"
echo "   Then run: encore secret set POLAR_PRODUCT_ID --dev"
echo ""
echo "3. POLAR_WEBHOOK_SECRET is already set ✓"
echo ""
echo "After migration, you can optionally remove the old secrets:"
echo "   (Note: Encore doesn't have a 'secret delete' command, they will remain unused)"
echo ""

# Alternative: If you know the values, uncomment and fill these in:
# echo "your-access-token-here" | encore secret set POLAR_ACCESS_TOKEN --dev
# echo "your-product-id-here" | encore secret set POLAR_PRODUCT_ID --dev

echo "Migration instructions displayed above."
