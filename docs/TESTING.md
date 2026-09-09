# 测试与验收 / Testing

## Automated / 自动化

```bash
go vet ./...
go test ./...
go test -race ./...
go build ./cmd/server
python scripts/test_release_export.py
python scripts/release_export.py --check-only
```

Race testing needs a supported platform/C toolchain; CI uses Ubuntu. The release export check validates the publication surface, not all private legacy material. `--check-tracked` additionally rejects tracked files outside the allowlist; use it only in the clean publication repository.

Required behavior coverage: anonymous mode with unset/empty token and no token file; both listeners and UI auto-connect without credentials; public auth-mode discovery; configured valid-token gates; nonempty invalid token or configured missing/unreadable/empty token file fails startup (no anonymous fallback); missing/invalid request token when auth is enabled; malformed request and unknown fields; range limits; unsupported band/mode; missing shielded acknowledgment; task cancellation/timeouts; process errors; redaction; pagination; observation deletion; no startup jobs; explicit demo labeling; graceful shutdown. 必须覆盖鉴权、参数边界、取消、错误和脱敏，不以HTTP200替代语义断言。

## Container smoke / 容器验收

Test both no-token and token-enabled configurations using the runtime target and no RF devices. For the authenticated configuration use a disposable random token. Confirm healthy liveness, successful authenticated status, unauthorized status rejection, embedded UI availability, non-root UID, read-only root and no privileged/host networking. Use `bash scripts/smoke.sh` with local `TOKEN`, `BASE` (default `http://127.0.0.1:18083`) and `API_BASE` (default `http://127.0.0.1:8083`) environment values. CI may override both URLs. The script probes both health endpoints and checks the API-only backend: `/` returns 404, `/api/v1/status` without a token returns 401 and with a valid token returns 200. It also checks frontend HTML and same-origin authenticated routes. The script is read-only and creates no job. In a separate anonymous run, require `data.required=false`, 200 for unauthenticated status on both ports, automatic UI connection without a token form, and no startup jobs.

冒烟测试默认同时检查前端18083与后端8083，验证后端不提供HTML、两端鉴权及健康状态。CI自定义映射时须同时覆盖BASE和API_BASE；没有启动任何RF任务。

Import Postman collection, set local variables, run all read-only requests. Tokens in Postman must be local/current values, not synced/shared initial values. The collection skips mutating requests by default. Explicit allowMutations=true permits the guarded demo-only lifecycle examples after a successful status request.

## UI manual / 前端验收

Check desktop/mobile layouts, Chinese/English switch, anonymous auto-connect, configured-token login and authentication errors, visible mode/restriction, status/capabilities, validation, one demo job, cancellation, empty/error states and redacted observations. Verify token is never placed in URLs or exported screenshots. Record browser/version and viewport.

## RF acceptance / 射频验收（单独）

A physical test is a separate gated activity: owned test SIM/terminal, documented shielding check, exact SDR/driver versions, explicit hardware mapping, bounded duration, acknowledged mode, expected synthetic/lab identifiers only, cancellation and shutdown verification. Keep RF captures private; publish only sanitized counts/results. If not run, report **not tested** rather than claiming full end-to-end acceptance.

## Evidence / 证据

Release notes should list commands, OS/Go version, image digest, pass/fail and skipped tests. A source build, Docker image build, healthy HTTP server, UI check and RF acceptance are distinct results. 本文是验收计划，不是测试已通过的记录。

Scan-first regression: `node scripts/test_web_workflow.js` drives the actual UI handlers with synthetic API state through scan, stop, select IMSI, capture, stop, select SMS, and clear/invalidation. Go catalog/API tests cover matching provenance, busy state, source mode, restart, deduplication and legacy history. These tests do not exercise radio hardware.
