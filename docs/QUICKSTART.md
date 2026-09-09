# 快速开始 / Quick start

1. Read the restriction in the root README. Keep the first run in `demo`; no RF device mapping is required. 首次使用 demo，不映射硬件。
2. Generate a random token file and set its owner to container UID 10001 as shown in README. Save the token in a password manager before changing ownership. 将 Token 保存到密码管理器，不要粘贴到共享命令或日志中。
3. Export an absolute `GSMSNIFFER_TOKEN_PATH`, then run `docker compose -f deploy/docker/compose.yml up -d --build` from the repository root. `.env` is optional; explicitly pass `--env-file .env` if used.
4. Check `docker compose -f deploy/docker/compose.yml ps` and `curl --fail http://127.0.0.1:8080/healthz`. A green health check proves HTTP liveness, not RF readiness. 健康检查只证明 HTTP 服务存活。
5. Open the console and enter the token. Query capabilities/status before creating a demo job. Status must clearly indicate `demo`. 模拟数据不可作为现场测量结果。
6. Finish with `docker compose -f deploy/docker/compose.yml down`. Do not add `-v` unless data deletion is intended. 停止服务默认保留数据卷。

For a prebuilt image, first confirm that the named version exists in your account, then set `GSMSNIFFER_IMAGE=addxemmm/gsmsniffer:2.0.0`, run `docker compose ... pull`, and `docker compose ... up -d --no-build`. These examples do not assert that a public image currently exists.

For local development use Go 1.26, `go test ./...`, and `go run ./cmd/server`, with a readable token file and `GSMSNIFFER_ADDR=127.0.0.1:8080`. Windows PowerShell uses `$env:GSMSNIFFER_TOKEN_FILE='C:\absolute\path\api_token'`.
