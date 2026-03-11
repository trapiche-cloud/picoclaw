#!/bin/sh
set -e

# Run the launcher (web console) on port 3000.
# The launcher auto-starts the picoclaw gateway as a subprocess
# and serves the dashboard UI on /.
exec picoclaw-launcher -port 3000 -public -no-browser
