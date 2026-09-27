#!/usr/bin/env sh
# Example: fetch the title version list, then prove the cheap 304 path.
set -e
BASE="${1:-http://localhost:8471}"
echo "== GET /tagaya/hac_versionlist =="
ETAG=$(curl -s -D - -o /tmp/vl.json "$BASE/tagaya/hac_versionlist" | tr -d '\r' | awk 'tolower($1)=="etag:"{print $2}')
cat /tmp/vl.json; echo
echo "etag: $ETAG"
echo "== re-request with If-None-Match (expect 304) =="
curl -s -o /dev/null -w "status: %{http_code}\n" -H "If-None-Match: $ETAG" "$BASE/tagaya/hac_versionlist"
