# 快速开始 / Quick start

1. Read the restriction in the root README. Keep the first run in `demo`; no RF device mapping is required. 首次使用 demo，不映射硬件。
2. Generate a random token file and set its owner to container UID 10001 as shown in README. Save the token in a password manager before changing ownership. 将 Token 保存到密码管理器，不要粘贴到共享命令或日志中。
3. Export an absolute `GSMSNIFFER_TOKEN_PATH`, then run `docker compose -f deploy/docker/compose.yml up -d --build` from the repository root. `.env` is optional; explicitly pass `--env-file .env` if used.
4. Check `docker compose -f deploy/docker/compose.yml ps` and both probes: `curl --fail http://127.0.0.1:18083/healthz` and `curl --fail http://127.0.0.1:8083/healthz`. A green health check proves HTTP liveness, not RF readiness. 健康检查只证明 HTTP 服务存活。
5. Open `http://127.0.0.1:18083` and enter the token; the API-only backend is `http://127.0.0.1:8083/api/v1`. 前端18083与后端8083默认同时启用，前端仍调用同源API。 Query capabilities/status before creating a demo job. Status must clearly indicate `demo`. 模拟数据不可作为现场测量结果。
6. Finish with `docker compose -f deploy/docker/compose.yml down`. Do not add `-v` unless data deletion is intended. 停止服务默认保留数据卷。

Use the single prebuilt image / 使用唯一预构建版本：

```bash
export GSMSNIFFER_IMAGE=addxemmm/gsmsniffer:2.1
docker compose -f deploy/docker/compose.yml pull
docker compose -f deploy/docker/compose.yml up -d --no-build
```

The image includes gr-gsm/tshark but defaults to demo without USB mapping. Compose uses `restart: "no"`; starting it here does not enable boot startup. One tag serves both modes; hardware integration requires the separately reviewed override in [DEPLOY](DEPLOY.md).

同一2.1镜像包含完整依赖，默认仅demo；不需要额外镜像标签。服务不开机自启，镜像存在不代表真实射频验收完成。

For local development use Go 1.26, `go test ./...`, and `go run ./cmd/server`, with a readable token file and both `GSMSNIFFER_ADDR=127.0.0.1:18083` and `GSMSNIFFER_API_ADDR=127.0.0.1:8083` for loopback-only local listeners. 本地开发建议两个ADDR都指定loopback；应用默认分别为 `:18083` 和 `:8083`。 Windows PowerShell uses `$env:GSMSNIFFER_TOKEN_FILE='C:\absolute\path\api_token'`.
