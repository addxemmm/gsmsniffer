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

Required behavior coverage: missing/invalid token; malformed request and unknown fields; range limits; unsupported band/mode; missing shielded acknowledgment; task cancellation/timeouts; process errors; redaction; pagination; observation deletion; no startup jobs; explicit demo labeling; graceful shutdown. 必须覆盖鉴权、参数边界、取消、错误和脱敏，不以HTTP200替代语义断言。

## Container smoke / 容器验收

Use a disposable random token, runtime target and no RF devices. Confirm healthy liveness, successful authenticated status, unauthorized status rejection, embedded UI availability, non-root UID, read-only root and no privileged/host networking. Use `bash scripts/smoke.sh` with local `BASE` and `TOKEN` environment values. The script is read-only and creates no job.

Import Postman collection, set local variables, run all read-only requests. Tokens in Postman must be local/current values, not synced/shared initial values. The collection skips mutating requests by default. Explicit allowMutations=true permits the guarded demo-only lifecycle examples after a successful status request.

## UI manual / 前端验收

Check desktop/mobile layouts, Chinese/English switch, authentication errors, visible mode/restriction, status/capabilities, validation, one demo job, cancellation, empty/error states and redacted observations. Verify token is never placed in URLs or exported screenshots. Record browser/version and viewport.

## RF acceptance / 射频验收（单独）

A physical test is a separate gated activity: owned test SIM/terminal, documented shielding check, exact SDR/driver versions, explicit hardware mapping, bounded duration, acknowledged mode, expected synthetic/lab identifiers only, cancellation and shutdown verification. Keep RF captures private; publish only sanitized counts/results. If not run, report **not tested** rather than claiming full end-to-end acceptance.

## Evidence / 证据

Release notes should list commands, OS/Go version, image digest, pass/fail and skipped tests. A source build, Docker image build, healthy HTTP server, UI check and RF acceptance are distinct results. 本文是验收计划，不是测试已通过的记录。
