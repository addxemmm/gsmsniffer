#!/usr/bin/env bash
# Read-only dual-listener smoke test; never starts jobs or touches RF devices.
set -euo pipefail
BASE="${BASE:-http://127.0.0.1:18083}"
API_BASE="${API_BASE:-http://127.0.0.1:8083}"
AUTH_REQUIRED="${AUTH_REQUIRED:-true}"
auth=()
case "$AUTH_REQUIRED" in
  true) : "${TOKEN:?Set TOKEN locally; never commit it}"; auth=(-H "Authorization: Bearer $TOKEN"); expected=401 ;;
  false) expected=200 ;;
  *) echo 'AUTH_REQUIRED must be true or false' >&2; exit 1 ;;
esac
BASE="${BASE%/}"
API_BASE="${API_BASE%/}"
expect_status() {
  local expected="$1" actual
  shift
  actual=$(curl --silent --show-error --connect-timeout 5 --max-time 15 \
    --output /dev/null --write-out '%{http_code}' "$@")
  if [ "$actual" != "$expected" ]; then
    printf 'Expected HTTP %s, got %s\n' "$expected" "$actual" >&2
    return 1
  fi
}
for endpoint in "$BASE" "$API_BASE"; do
  curl --fail --silent --show-error "$endpoint/api/v1/auth" | python3 -c 'import json,sys; assert json.load(sys.stdin)["data"]["required"] == (sys.argv[1] == "true")' "$AUTH_REQUIRED"
done
# Both listeners must be alive; backend root must not expose frontend HTML.
expect_status 200 "$BASE/healthz"
expect_status 200 "$API_BASE/healthz"
expect_status 404 "$API_BASE/"
expect_status "$expected" "$BASE/api/v1/status"
expect_status "$expected" "$API_BASE/api/v1/status"
expect_status 200 "${auth[@]}" "$API_BASE/api/v1/status"
for route in status capabilities jobs 'observations?kind=frequencies&limit=1&offset=0'; do
  curl --fail --silent --show-error --connect-timeout 5 --max-time 15 \
    "${auth[@]}" "$BASE/api/v1/$route" \
    | python3 -c 'import json,sys; r=json.load(sys.stdin); assert all(k in r for k in ("code","message","data","request_id")); assert r["code"] == "ok"; assert r["request_id"]'
done
curl --fail --silent --show-error --connect-timeout 5 --max-time 15 "$BASE/" \
  | python3 -c 'import sys; assert "<html" in sys.stdin.read().lower()'
printf 'Read-only frontend/backend smoke checks passed. RF hardware was not exercised.\n'
