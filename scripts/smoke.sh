#!/usr/bin/env bash
# Read-only HTTP smoke test; never starts jobs or touches RF devices.
set -euo pipefail
BASE="${BASE:-http://127.0.0.1:8080}"
: "${TOKEN:?Set TOKEN locally; never commit it}"
curl --fail --silent --show-error "$BASE/healthz" >/dev/null
code=$(curl --silent --show-error --output /dev/null --write-out '%{http_code}' "$BASE/api/v1/status")
test "$code" = 401
for route in status capabilities jobs 'observations?kind=frequencies&limit=1&offset=0'; do
  curl --fail --silent --show-error -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/$route" \
    | python3 -c 'import json,sys; r=json.load(sys.stdin); assert all(k in r for k in ("code","message","data","request_id")); assert r["request_id"]'
done
curl --fail --silent --show-error "$BASE/" | grep -qi '<html'
printf 'Read-only demo smoke checks passed. RF hardware was not exercised.\n'
