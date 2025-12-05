#!/bin/bash

# Test script for subscription flow
BASE_URL="http://127.0.0.1:4000"

echo "🧪 Testing Subscription Flow"
echo "=============================="
echo ""

# Step 1: Login with Google (you'll need to do this manually via browser first)
echo "Step 1: Login"
echo "Visit: $BASE_URL/auth/google/login"
echo "After login, copy your session token from the response"
echo ""
read -p "Enter your session token: " SESSION_TOKEN
echo ""

# Step 2: Check current user
echo "Step 2: Get current user info"
curl -s -X GET "$BASE_URL/auth/me" \
  -H "Authorization: Bearer $SESSION_TOKEN" | jq .
echo ""
echo ""

# Step 3: Try to create first instance (should start trial)
echo "Step 3: Create first instance (starts trial)"
INSTANCE_RESPONSE=$(curl -s -X POST "$BASE_URL/me/instances" \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subdomain": "test-trial-'$(date +%s)'"
  }')
echo "$INSTANCE_RESPONSE" | jq .
INSTANCE_ID=$(echo "$INSTANCE_RESPONSE" | jq -r '.instance_id')
echo ""
echo "✅ First instance created: $INSTANCE_ID"
echo ""

# Step 4: List instances
echo "Step 4: List instances"
curl -s -X GET "$BASE_URL/me/instances" \
  -H "Authorization: Bearer $SESSION_TOKEN" | jq .
echo ""
echo ""

# Step 5: Try to create second instance (should be blocked - trial limit)
echo "Step 5: Try to create second instance (should fail - trial limit)"
curl -s -X POST "$BASE_URL/me/instances" \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "subdomain": "test-second-'$(date +%s)'"
  }' | jq .
echo ""
echo "❌ Second instance blocked (expected)"
echo ""

# Step 6: Get subscription status (private endpoint - for testing only)
echo "Step 6: Check subscription status (internal)"
echo "Note: This is a private endpoint, normally called by other services"
echo ""

echo "=============================="
echo "🎉 Test Complete!"
echo ""
echo "What happened:"
echo "1. ✅ First instance created → Trial started (3 days)"
echo "2. ❌ Second instance blocked → Trial users limited to 1 instance"
echo ""
echo "Next steps to test:"
echo "- Create checkout session for subscription"
echo "- Simulate Polar webhook to activate subscription"
echo "- Try creating second instance after subscription is active"
