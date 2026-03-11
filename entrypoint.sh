#!/bin/sh
set -e

# Start picoclaw gateway in background
picoclaw gateway &

# Start Caddy in foreground on port 3000
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
