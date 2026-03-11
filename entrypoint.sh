#!/bin/sh
set -e

# Start the launcher (web dashboard) on port 18800
picoclaw-launcher -port 18800 -public -no-browser &

# Wait for launcher to start (it auto-starts the gateway on 18790)
sleep 2

# Caddy on port 3000 unifies both:
#   /pico/* → gateway:18790  (WebSocket)
#   /*      → launcher:18800 (dashboard)
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
