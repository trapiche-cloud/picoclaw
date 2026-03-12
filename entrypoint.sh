#!/bin/sh
set -e

CONFIG="/home/picoclaw/.picoclaw/config.json"

# Patch config from environment variables on every startup.
# This ensures settings survive container restarts / redeployments.

# Set default model (e.g. PICOCLAW_DEFAULT_MODEL=openai/gpt-5.2)
if [ -n "$PICOCLAW_DEFAULT_MODEL" ]; then
  sed -i "s|\"model\": \"\"|\"model\": \"$PICOCLAW_DEFAULT_MODEL\"|" "$CONFIG"
fi

# Set OpenAI API key
if [ -n "$OPENAI_API_KEY" ]; then
  # Find the openai model_list entry and set its api_key
  sed -i "/\"model\": \"openai\//{ n; n; s|\"api_key\": \"[^\"]*\"|\"api_key\": \"$OPENAI_API_KEY\"|; }" "$CONFIG"
fi

# Enable Telegram with token
if [ -n "$PICOCLAW_TELEGRAM_TOKEN" ]; then
  sed -i "/\"telegram\"/{n;s/\"enabled\": false/\"enabled\": true/}" "$CONFIG"
  sed -i "s|\"token\": \"8748831181:[^\"]*\"|\"token\": \"$PICOCLAW_TELEGRAM_TOKEN\"|" "$CONFIG"
fi

# Start the launcher (web dashboard) on port 18800
picoclaw-launcher -port 18800 -public -no-browser &

# Wait for launcher to start (it auto-starts the gateway on 18790)
sleep 2

# Caddy on port 3000 unifies both:
#   /pico/* → gateway:18790  (WebSocket)
#   /*      → launcher:18800 (dashboard)
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
