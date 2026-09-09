#!/usr/bin/env bash
# Disposable containers: both auth modes, no devices and no RF jobs.
set -euo pipefail
image="${1:?Supply the locally built image}"
temp=$(mktemp -d)
name="gsmsniffer-ci-${GITHUB_RUN_ID:-$$}"
cleanup() { docker rm -f "$name" >/dev/null 2>&1 || true; rm -f "$temp/token" "$temp/invalid.log"; rmdir "$temp"; }
trap cleanup EXIT
TOKEN=$(openssl rand -hex 32)
if [ -n "${GITHUB_ACTIONS:-}" ]; then echo "::add-mask::$TOKEN"; fi
printf '%s\n' "$TOKEN" > "$temp/token"
chmod 0444 "$temp/token"
for required in true false; do
  args=()
  if [ "$required" = true ]; then
    args=(--mount "type=bind,src=$temp/token,dst=/run/secrets/api_token,readonly" -e GSMSNIFFER_TOKEN_FILE=/run/secrets/api_token)
  fi
  docker run -d --name "$name" --read-only --cap-drop ALL --security-opt no-new-privileges \
    --tmpfs /tmp:rw,noexec,nosuid,size=64m \
    --tmpfs /var/lib/gsmsniffer:rw,uid=10001,gid=10001,mode=0700 \
    -p 127.0.0.1:18083:18083 -p 127.0.0.1:8083:8083 "${args[@]}" "$image"
  for attempt in $(seq 1 30); do
    if curl --fail --silent http://127.0.0.1:18083/healthz >/dev/null; then break; fi
    sleep 1
  done
  TOKEN="$TOKEN" AUTH_REQUIRED="$required" BASE=http://127.0.0.1:18083 API_BASE=http://127.0.0.1:8083 bash scripts/smoke.sh
  docker rm -f "$name" >/dev/null
done
# An explicitly configured broken credential must not silently disable auth.
if docker run --rm --network none -e GSMSNIFFER_TOKEN_FILE=/missing-token "$image" >"$temp/invalid.log" 2>&1; then
  echo 'Invalid configured token file unexpectedly succeeded' >&2
  exit 1
fi
grep -q 'read token file' "$temp/invalid.log"
env -u GSMSNIFFER_TOKEN_PATH -u GSMSNIFFER_TOKEN docker compose -f deploy/docker/compose.yml config --quiet
echo 'Anonymous and token-enabled containers passed; invalid file failed closed; base Compose needs no secret.'
