# 快速开始 / Quick start

## Scan-first workflow / 先扫频点再扫描 IMSI、SMS

1. 打开默认的“扫描工作台”，选择 GSM900 或 DCS1800，确认屏蔽条件后开始频点扫描。
2. 扫描结果实时显示在同一页，可按扫描批次筛选。扫描完成或点击“停止任务”后，频点行的 IMSI / SMS 按钮才可选择。
3. 点击频点行的 IMSI 或 SMS，自动带入频段、频率和来源扫描任务；这些参数不再手动填写。
4. 设置时长，重新确认屏蔽条件，显式开始采集。所选频点的结果和停止按钮仍在工作台内；可在“观测数据”查看完整历史。
5. 一次只运行一个任务。完成/停止 IMSI 后，可从同一频点选择 SMS，反之亦然；不自动启动下一任务。

No frequencies means no capture. Scan first, then choose a selectable result. The backend verifies `scan_job_id`, band, frequency and the current runtime source; a forged/manual, cleared, failed-scan or unavailable historical selection is rejected. Running scan results are visible but not selectable until finished or cancelled. Unlinked legacy observations remain readable but require a new scan before reuse. The catalog is deduplicated per scan/channel and bounded by retained observations.

无频点时不允许进入采集；清空观测会使已选频点失效。旧版本没有扫描来源关联的频点需重新扫描。DEMO 始终是合成数据，不代表真实接收；IMSI 保持脱敏，SMS 仅保留事件、不保存正文。硬件模式与屏蔽条件不会因为点击频点而自动改变。


1. Read the restriction in the root README. Keep the first run in `demo`; no RF device mapping is required. 首次使用 demo，不映射硬件。
2. Choose authentication: for a trusted isolated LAN, set `GSMSNIFFER_TOKEN` empty (including any Compose environment file) and do not configure a token file. The console then connects automatically. To require login, follow the optional file-secret overlay in [Deployment](DEPLOY.md#2-optional-authentication--可选鉴权). 免登录时所有能访问管理端口的人均可管理任务及数据；请限制来源。
3. For the no-token deployment, run `docker compose -f deploy/docker/compose.yml up -d --build` from the repository root. `.env` is optional; explicitly pass `--env-file .env` if used.
4. Check `docker compose -f deploy/docker/compose.yml ps` and both probes: `curl --fail http://127.0.0.1:18083/healthz` and `curl --fail http://127.0.0.1:8083/healthz`. A green health check proves HTTP liveness, not RF readiness. 健康检查只证明 HTTP 服务存活。
5. Open `http://127.0.0.1:18083` and allow automatic connection (or enter the token when authentication is configured); the API-only backend is `http://127.0.0.1:8083/api/v1`. 前端18083与后端8083默认同时启用，前端仍调用同源API。 Query capabilities/status before creating a demo job. Status must clearly indicate `demo`. 模拟数据不可作为现场测量结果。
6. Finish with `docker compose -f deploy/docker/compose.yml down`. Do not add `-v` unless data deletion is intended. 停止服务默认保留数据卷。

Use the single prebuilt image / 使用唯一预构建版本：

```bash
export GSMSNIFFER_IMAGE=addxemmm/gsmsniffer:2.1
docker compose -f deploy/docker/compose.yml pull
docker compose -f deploy/docker/compose.yml up -d --no-build
```

The image includes gr-gsm/tshark but defaults to demo without USB mapping. Compose uses `restart: "no"`; starting it here does not enable boot startup. One tag serves both modes; hardware integration requires the separately reviewed override in [DEPLOY](DEPLOY.md).

同一2.1镜像包含完整依赖，默认仅demo；不需要额外镜像标签。服务不开机自启，镜像存在不代表真实射频验收完成。

For trusted LAN HTTP without a domain or certificate, follow [Deployment](DEPLOY.md#trusted-lan-http--受信任局域网-http): explicitly set both `GSMSNIFFER_BIND=0.0.0.0` and `GSMSNIFFER_API_BIND=0.0.0.0` in the existing deployment environment file, then recreate the same Compose project. Visit `http://HOST:18083`, with `HOST` replaced by the server's RFC1918 private IPv4 address. HTTP 明文传输 Token 与数据，仅限可信局域网并限制来源，禁止公网端口转发；默认 Compose 仍绑定 loopback。

For local development use Go 1.26, `go test ./...`, and `go run ./cmd/server`, with an optional readable token file and both `GSMSNIFFER_ADDR=127.0.0.1:18083` and `GSMSNIFFER_API_ADDR=127.0.0.1:8083` for loopback-only local listeners. 本地开发建议两个ADDR都指定loopback；应用默认分别为 `:18083` 和 `:8083`。 Windows PowerShell uses `$env:GSMSNIFFER_TOKEN_FILE='C:\absolute\path\api_token'`.
