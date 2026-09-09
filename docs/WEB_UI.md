# Web 控制台 / Web console

## Scan-first workflow / 先扫频点再扫描 IMSI、SMS

1. 打开默认的“扫描工作台”，选择 GSM900 或 DCS1800，确认屏蔽条件后开始频点扫描。
2. 扫描结果实时显示在同一页，可按扫描批次筛选。扫描完成或点击“停止任务”后，频点行的 IMSI / SMS 按钮才可选择。
3. 点击频点行的 IMSI 或 SMS，自动带入频段、频率和来源扫描任务；这些参数不再手动填写。
4. 设置时长，重新确认屏蔽条件，显式开始采集。所选频点的结果和停止按钮仍在工作台内；可在“观测数据”查看完整历史。
5. 一次只运行一个任务。完成/停止 IMSI 后，可从同一频点选择 SMS，反之亦然；不自动启动下一任务。

No frequencies means no capture. Scan first, then choose a selectable result. The backend verifies `scan_job_id`, band, frequency and the current runtime source; a forged/manual, cleared, failed-scan or unavailable historical selection is rejected. Running scan results are visible but not selectable until finished or cancelled. Unlinked legacy observations remain readable but require a new scan before reuse. The catalog is deduplicated per scan/channel and bounded by retained observations.

无频点时不允许进入采集；清空观测会使已选频点失效。旧版本没有扫描来源关联的频点需重新扫描。DEMO 始终是合成数据，不代表真实接收；IMSI 保持脱敏，SMS 仅保留事件、不保存正文。硬件模式与屏蔽条件不会因为点击频点而自动改变。


The embedded console is served at `/` on port 18083. The independent API-only backend defaults to port 8083 and does not serve the console (`/` returns 404). It shares its origin with `/api/v1`, avoiding a separate web-server or cross-origin token configuration. Select Chinese or English and inspect status and capabilities first. The console queries public `/api/v1/auth`: no configured token means automatic connection without a login form; configured authentication requires a bearer token.

控制台默认18083，使用同源 `/api/v1` 接口；独立后端默认8083且只提供API，两个监听器默认启用。先确认服务模式、依赖状态及是否启用鉴权，再操作任务。未配置 Token 时免登录自动连接；任一可访问管理端口的人均可操作任务和数据。初次验收只运行 demo；演示频点和脱敏身份不是真实测量。

Recommended operator sequence:

1. Confirm the displayed mode. Check the shielded-room/enclosure restriction and owned test SIM/device scope.
2. Review available capabilities; dependency presence alone is not hardware acceptance.
3. Choose a supported band, job type, frequency and bounded duration. Acknowledge physical isolation only after checking it.
4. Start once, monitor the returned job ID, and use cancel to stop a running job.
5. Inspect redacted observations, then clear them when no longer needed. Avoid sharing screenshots containing tokens or unredacted data.
6. When using authentication, end the session and remove token access from a shared browser. In anonymous mode, disconnecting the page does not revoke API access. Do not paste tokens into issue reports.

若界面报错，先检查 HTTP 状态和 request_id、Token 文件、容器日志、`/healthz`，再检查能力接口。独立API监听器默认8083且已启用；界面仍走18083同源API，不依赖浏览器跨域访问8083。

## 局域网访问 / LAN access

仅在受信任局域网，可按 [部署指南](DEPLOY.md) 将前后端宿主绑定地址显式设为 `0.0.0.0`，然后访问 `http://HOST:18083`，其中 `HOST` 必须替换为服务器的 RFC1918 私有 IPv4 地址；不要求域名或证书。Token 仅保留在当前页面内存中，不写入浏览器持久存储。

The console allows HTTP token submission on loopback and literal RFC1918 private IPv4 addresses, and allows HTTPS. Public HTTP addresses and ordinary hostnames are rejected for token submission; a hostname resolving to a private address is not automatically treated as private. Use the server's private IPv4 address for LAN HTTP, or use HTTPS/an SSH tunnel for other access patterns.

**HTTP 会明文传输 Token 和数据，只适合可信局域网。** 限制端口来源并禁止公网端口转发；浏览器的地址检查不构成网络隔离。扩大监听范围不改变服务端可选鉴权配置，也不会启动 RF 任务；未配置 Token 时两端口均允许匿名管理。HTTP transmits tokens and data without encryption; restrict access to trusted clients. Browser checks do not replace firewall controls or authenticate the network.

A responsive layout and bilingual labels are UI features, not a certification of accessibility or RF operation; record browser and viewport in manual acceptance reports.
