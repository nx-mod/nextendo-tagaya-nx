#!/usr/bin/env sh
# nextendo-tagaya-nx — home-lab launch (title version list). Serves plain HTTP
# (TLS via the sni-router); versions.json ships in the repo (with an embedded
# fallback), so it runs straight from the repo.
set -e
echo "[tagaya] starting on default ports (TAGAYA_PORT 8471, DASH_PORT 8100)"
exec go run .
