# Web 控制台 / Web console

The embedded console is served at `/` on port 18083. The independent API-only backend defaults to port 8083 and does not serve the console (`/` returns 404). It shares its origin with `/api/v1`, avoiding a separate web-server or cross-origin token configuration. Select Chinese or English, enter your bearer token, and inspect status and capabilities first.

控制台默认18083，使用同源 `/api/v1` 接口；独立后端默认8083且只提供API，两个监听器默认启用。先确认服务模式、依赖状态和 Token，再操作任务。初次验收只运行 demo；演示频点和脱敏身份不是真实测量。

Recommended operator sequence:

1. Confirm the displayed mode. Check the shielded-room/enclosure restriction and owned test SIM/device scope.
2. Review available capabilities; dependency presence alone is not hardware acceptance.
3. Choose a supported band, job type, frequency and bounded duration. Acknowledge physical isolation only after checking it.
4. Start once, monitor the returned job ID, and use cancel to stop a running job.
5. Inspect redacted observations, then clear them when no longer needed. Avoid sharing screenshots containing tokens or unredacted data.
6. End the session and remove token access from a shared browser. Do not paste tokens into issue reports.

若界面报错，先检查 HTTP 状态和 request_id、Token 文件、容器日志、`/healthz`，再检查能力接口。独立API监听器默认8083且已启用；界面仍走18083同源API，不依赖浏览器跨域访问8083。

## 局域网访问 / LAN access

仅在受信任局域网，可按 [部署指南](DEPLOY.md) 将前后端宿主绑定地址显式设为 `0.0.0.0`，然后访问 `http://HOST:18083`，其中 `HOST` 必须替换为服务器的 RFC1918 私有 IPv4 地址；不要求域名或证书。Token 仅保留在当前页面内存中，不写入浏览器持久存储。

The console allows HTTP token submission on loopback and literal RFC1918 private IPv4 addresses, and allows HTTPS. Public HTTP addresses and ordinary hostnames are rejected for token submission; a hostname resolving to a private address is not automatically treated as private. Use the server's private IPv4 address for LAN HTTP, or use HTTPS/an SSH tunnel for other access patterns.

**HTTP 会明文传输 Token 和数据，只适合可信局域网。** 限制端口来源并禁止公网端口转发；浏览器的地址检查不构成网络隔离。扩大监听范围不会关闭 API 鉴权，也不会启动 RF 任务。HTTP transmits tokens and data without encryption; restrict access to trusted clients. Browser checks do not replace firewall controls or authenticate the network.

A responsive layout and bilingual labels are UI features, not a certification of accessibility or RF operation; record browser and viewport in manual acceptance reports.
