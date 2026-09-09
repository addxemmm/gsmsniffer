# GSM-SNIFFER 2.1 · Go laboratory console

## Scan-first workflow / 先扫频点再扫描 IMSI、SMS

1. 打开默认的“扫描工作台”，选择 GSM900 或 DCS1800，确认屏蔽条件后开始频点扫描。
2. 扫描结果实时显示在同一页，可按扫描批次筛选。扫描完成或点击“停止任务”后，频点行的 IMSI / SMS 按钮才可选择。
3. 点击频点行的 IMSI 或 SMS，自动带入频段、频率和来源扫描任务；这些参数不再手动填写。
4. 设置时长，重新确认屏蔽条件，显式开始采集。所选频点的结果和停止按钮仍在工作台内；可在“观测数据”查看完整历史。
5. 一次只运行一个任务。完成/停止 IMSI 后，可从同一频点选择 SMS，反之亦然；不自动启动下一任务。

No frequencies means no capture. Scan first, then choose a selectable result. The backend verifies `scan_job_id`, band, frequency and the current runtime source; a forged/manual, cleared, failed-scan or unavailable historical selection is rejected. Running scan results are visible but not selectable until finished or cancelled. Unlinked legacy observations remain readable but require a new scan before reuse. The catalog is deduplicated per scan/channel and bounded by retained observations.

无频点时不允许进入采集；清空观测会使已选频点失效。旧版本没有扫描来源关联的频点需重新扫描。DEMO 始终是合成数据，不代表真实接收；IMSI 保持脱敏，SMS 仅保留事件、不保存正文。硬件模式与屏蔽条件不会因为点击频点而自动改变。


[GitHub Releases](https://github.com/addxemmm/gsmsniffer/releases) · [Docker Hub](https://hub.docker.com/r/addxemmm/gsmsniffer) · [2.1 说明 / Release notes](docs/releases/2.1.md)

> **使用范围：仅限自有测试 SIM、自有终端，在屏蔽室或屏蔽箱内进行隔离实验。不得接入或采集公众移动网络及第三方通信。射频任务前应确认屏蔽有效、测试设备归属及现场操作条件。**
>
> **Use only your own test SIMs and terminals inside an RF-shielded room or shielded enclosure. Do not connect to or collect public-network or third-party communications. Verify isolation and device ownership before any RF job.**

一个 Go 标准库管理后端，内嵌中英双语控制台与版本化 REST API。默认 `demo` 仅生成模拟数据、不访问 RF 硬件；容器启动不自动创建任务。`shielded` 是明确启用的实验模式，不代表已经完成硬件验收。

A Go standard-library management backend with an embedded bilingual console and versioned REST API. Default `demo` uses synthetic data, without RF hardware access. Startup creates no jobs. Explicit `shielded` mode is for laboratory integration; it is not a claim of hardware validation.

## 能力 / Features

- 前端 `:18083` 保留同源 `/api/v1`；独立后端 `:8083` 默认启用且只提供 API、不提供界面。Both listeners are enabled by default: UI and same-origin API on `:18083`, API-only backend on `:8083`.
- 可选 Bearer Token：未配置 Token 时免登录自动连接；配置后要求登录，token 文件优先。Optional authentication: no configured token means anonymous access; configured credentials enable bearer authentication.
- `/api/v1` 标准包络、OpenAPI、Postman 断言。Versioned envelopes, OpenAPI and Postman checks.
- 有界扫描/采集任务、取消与状态查询；身份永久脱敏，SMS只保留事件、不保留明文正文。Bounded jobs and cancellation; identities are permanently masked and SMS is event-only, without plaintext bodies.
- 单一 Docker 镜像 `addxemmm/gsmsniffer:2.1` 内含 gr-gsm/tshark；默认非 root、只读根文件系统、无特权及 USB 映射。One complete image; RF device access and shielded mode remain opt-in.
- 前端18083、后端8083；Compose 重启策略为 `no`，不开机自启。No automatic service restart or startup jobs.
- 新源码白名单导出；旧日志、捕获、图片/PDF、历史仓库不进入发布包。Allowlisted publication excludes legacy data and history.

## 快速开始 / Quick start

Requires Docker Engine + Compose v2 on the Linux server. From the new source checkout:

```bash
# No token is required for a trusted local deployment.
export GSMSNIFFER_TOKEN='' # Also overrides any token in a Compose .env file.
unset GSMSNIFFER_TOKEN_FILE
docker compose -f deploy/docker/compose.yml up -d --build
curl --fail http://127.0.0.1:18083/healthz
curl --fail http://127.0.0.1:8083/healthz
```

Open `http://127.0.0.1:18083`; with no configured token, the console connects automatically without login. If a token is configured, enter it to connect. For trusted LAN access without a domain or certificate, explicitly set both `GSMSNIFFER_BIND=0.0.0.0` and `GSMSNIFFER_API_BIND=0.0.0.0` before recreating the container, then visit `http://HOST:18083`, replacing `HOST` with the server's RFC1918 private IPv4 address. HTTP transmits the bearer token and data without encryption: restrict access to trusted LAN clients and do not forward these ports to the Internet. An SSH tunnel (`ssh -L 18083:127.0.0.1:18083 USER@HOST`) or TLS reverse proxy remains available. The sample contains no server passwords or account tokens.

默认 Compose 将两个宿主端口绑定 loopback。仅在受信任局域网使用时，可显式将上述两个 BIND 变量设为 `0.0.0.0` 并重建容器：前端 `http://HOST:18083`，后端 `http://HOST:8083/api/v1`，`HOST` 替换为服务器的 RFC1918 私有 IPv4 地址，无需域名或证书。HTTP 会明文传输 Token 与数据，须限制可信网段访问，禁止公网端口转发。前端只允许 loopback、RFC1918 私有 IPv4 上的 HTTP 登录或 HTTPS 登录；普通主机名不视为私有地址。详细命令、Token 权限、清理和回滚见 [部署指南](docs/DEPLOY.md)。

```bash
# Development: Go 1.26; no Node build step required.
# Optional authentication: export GSMSNIFFER_TOKEN_FILE=/absolute/path/to/api_token
export GSMSNIFFER_ADDR=127.0.0.1:18083
export GSMSNIFFER_API_ADDR=127.0.0.1:8083
go test ./...
go run ./cmd/server
```

## 可选鉴权 / Optional authentication

未设置 `GSMSNIFFER_TOKEN_FILE`，且 `GSMSNIFFER_TOKEN` 未设置或为空时，前后端均免 Token 访问，控制台自动连接。**任何能访问任一管理端口的人均可读取/删除结果、创建/取消任务；免登录模式仅限可信局域网。** 网络绑定不是访问控制，请限制来源并禁止公网端口转发。

Set a valid `GSMSNIFFER_TOKEN` (at least 32 characters), or preferably use `deploy/docker/compose.auth.yml` with an absolute `GSMSNIFFER_TOKEN_PATH`, to require login on both ports. A configured missing/unreadable/empty token file or a nonempty invalid token causes startup failure; it never silently enables anonymous access. `GET /api/v1/auth` publicly reports only `data.required`.

启用文件鉴权的完整命令见 [部署指南](docs/DEPLOY.md#2-optional-authentication--可选鉴权)。基础 Compose 不再要求密钥文件；已有文件鉴权部署升级时须继续加载 `compose.auth.yml`，避免意外关闭鉴权。

## 文档 / Documentation

[文档索引](docs/README.md) · [快速开始](docs/QUICKSTART.md) · [架构](docs/ARCHITECTURE.md) · [API](docs/API.md) · [Web UI](docs/WEB_UI.md) · [部署](docs/DEPLOY.md) · [迁移](docs/MIGRATION.md) · [测试](docs/TESTING.md) · [发布](docs/RELEASING.md) · [安全](SECURITY.md)

## 发布与许可 / Publication and licensing

当前版本统一为 **`2.1`**，Docker Hub 仅使用 **`addxemmm/gsmsniffer:2.1`**。同一完整依赖镜像用于 demo 和显式 shielded 模式，不再发布独立依赖标签或提交哈希标签。默认前端18083、后端8083，`restart: "no"`。拉取和部署方式见 [快速开始](docs/QUICKSTART.md)。

Current release uses one Docker Hub tag, **`addxemmm/gsmsniffer:2.1`**, including gr-gsm/tshark. Demo remains the default, without hardware access or automatic jobs. Exact source revisions and image digests are recorded as release evidence rather than additional image tags. The [2.0.0 acceptance record](docs/releases/2.0.0.md) is historical, not current deployment guidance.

`2.0.0` 是破坏性改造版本，不保留旧工具的明文IMSI/SMS展示或导出兼容。No raw IMSI/SMS display or export compatibility is retained. 发布流程不自动登录实验服务器，不自动启动射频任务。旧工作目录与 Git 历史不得直接推入新公开项目。新增部分和外部运行组件的许可边界见 [NOTICE](NOTICE.md) 与 [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES.md)；不将未核实许可的旧代码重新声明为 MIT。
