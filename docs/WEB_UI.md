# Web 控制台 / Web console

The embedded console is served at `/` on port 8080. It shares its origin with `/api/v1`, avoiding a separate web-server or cross-origin token configuration. Select Chinese or English, enter your bearer token, and inspect status and capabilities first.

控制台使用同源接口。先确认服务模式、依赖状态和 Token，再操作任务。初次验收只运行 demo；演示频点和脱敏身份不是真实测量。

Recommended operator sequence:

1. Confirm the displayed mode. Check the shielded-room/enclosure restriction and owned test SIM/device scope.
2. Review available capabilities; dependency presence alone is not hardware acceptance.
3. Choose a supported band, job type, frequency and bounded duration. Acknowledge physical isolation only after checking it.
4. Start once, monitor the returned job ID, and use cancel to stop a running job.
5. Inspect redacted observations, then clear them when no longer needed. Avoid sharing screenshots containing tokens or unredacted data.
6. End the session and remove token access from a shared browser. Do not paste tokens into issue reports.

若界面报错，先检查 HTTP 状态和 request_id、Token 文件、容器日志、`/healthz`，再检查能力接口。直接 API listener默认关闭，界面无需8083。私有远程部署通过 SSH 隧道访问，不将 Token 暴露给普通 HTTP 公网连接。

A responsive layout and bilingual labels are UI features, not a certification of accessibility or RF operation; record browser and viewport in manual acceptance reports.
