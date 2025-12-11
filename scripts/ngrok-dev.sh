#!/bin/bash

# Start ngrok tunnel for local development
# This exposes 127.0.0.1:8080 to receive Polar webhooks

# Optional: Set your custom ngrok URL here (e.g., "https://your-domain.ngrok.dev")
# Get a free static domain from: https://dashboard.ngrok.com/cloud-edge/domains
NGROK_URL="${NGROK_URL:-}"

echo "Starting ngrok tunnel for 127.0.0.1:8080..."
echo ""

if [ -n "$NGROK_URL" ]; then
  echo "Using custom URL: $NGROK_URL"
  echo "Your webhook endpoint: $NGROK_URL/api/webhooks/polar"
  ngrok http 127.0.0.1:8080 --url="$NGROK_URL"
else
  echo "Using random ngrok URL (set NGROK_URL env var for custom URL)"
  echo "Your webhook endpoint will be: https://YOUR-NGROK-URL.ngrok.io/api/webhooks/polar"
  ngrok http 127.0.0.1:8080
fi

echo ""
echo "Configure this URL in your Polar webhook settings:"
echo "https://polar.sh/maintainer/ORGANIZATION/settings/webhooks"
echo ""
