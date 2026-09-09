# 2.1 迁移 / Migration

This is a breaking modernization release, not an in-place patch of the historical tool. Raw IMSI/SMS functionality is intentionally not preserved: identities are permanently masked and SMS output contains events only, not message bodies. 身份永久脱敏，短信仅事件；旧明文显示/导出不兼容，未提供解脱敏开关。

| Legacy / 旧版 | 2.x |
|---|---|
| Python management scripts | Go management server with embedded UI |
| Ad-hoc endpoints | Versioned `/api/v1`, optional authentication, envelope and OpenAPI |
| Implicit local state | Explicit runtime data directory and bounded jobs |
| Raw identity/SMS handling | Permanently masked identities; SMS events only, no plaintext body storage/export |
| Hardware-oriented startup | Default demo; no startup job |
| Workspace logs/images/history | Excluded from public source and image context |

1. Stop legacy experiments, retain a private rollback copy, and inventory secrets/data without exporting their contents. 停止旧任务，私下保留回滚副本。
2. Do not reuse the legacy `.git` history for the new public project; deletion in a later commit does not erase prior secrets. 不直接推送旧 Git 历史。
3. Run the allowlist export and security scan in `scripts/release_export.py`. Review every exported filename and content before initializing a clean Git repository.
4. Choose anonymous access for a trusted restricted LAN, or configure a new random token to require login. Do not import old logs, raw SMS, subscriber identities or arbitrary scripts into the new runtime.
5. Run tests and a demo-only container smoke test. Rewrite integrations to the v1 schema; old endpoint compatibility is not promised.
6. Complete separate shielded hardware acceptance before any real laboratory job.
7. Rollback by stopping the new container and restoring the private legacy deployment, only within the same isolated test conditions. Do not start both against the same SDR device.

New license declarations do not retroactively relicense upstream code. Review NOTICE and third-party obligations before publication. 新的管理代码与原工作目录许可边界分离。

## 2.1 single image and ports / 单镜像与端口

Current defaults enable UI/same-origin API `:18083` and an API-only backend `:8083`. Configure host mappings through `GSMSNIFFER_PORT` (18083) and `GSMSNIFFER_API_PORT` (8083), update tunnels/client URLs, and rerun dual-listener smoke checks. `compose.api.yml` is a no-op compatibility overlay, not an opt-in switch. File authentication now requires the optional `compose.auth.yml` overlay; loading the base file alone with no environment token enables anonymous access.

当前源码默认同时启用前端18083和后端8083；前端保留同源API，后端根路径返回404。迁移时更新宿主映射、SSH隧道、Postman与探活配置；若要沿用 Token 文件，必须加载 `compose.auth.yml`。升级后使用 `addxemmm/gsmsniffer:2.1`，完整依赖已包含在唯一镜像中；不再选单独的依赖或提交标签。`compose.shielded.yml` 只负责显式模式与设备权限覆盖。保留Token与数据卷，确认 `restart: "no"`，不开机自启。2.0.0文档保留历史事实，不作为当前配置依据。
