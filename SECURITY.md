# Security / 安全

Use only owned test SIMs/terminals within an RF-shielded room or enclosure. Default demo mode, explicit job acknowledgment and duration limits complement physical isolation; none replaces it.

运行默认：未配置 Token 时匿名访问；配置有效 Token 后启用 Bearer 鉴权，token 文件优先。身份永久脱敏与短信仅事件、不自动启动任务、容器非root/只读/无特权、基础 Compose 管理端口绑定loopback。不要把 Token、SSH 密码、IMSI、短信、RF捕获或旧Git历史上传至公开仓库。

Report suspected vulnerabilities privately to the repository owner using GitHub private vulnerability reporting when enabled. If it is unavailable, request a private contact without posting the exploit data or secrets publicly. Include affected version, environment, minimal synthetic reproduction, impact and request IDs. Never attach actual subscriber traffic.

If a secret is exposed, revoke/rotate it first, stop affected sessions, preserve a private incident record, then remove it from current content and history. A clean commit does not invalidate an exposed secret. Rebuild affected images and inspect image layers/artifacts where appropriate.

Before remote deployment: use an SSH tunnel/TLS management proxy, or explicitly enable HTTP only for trusted LAN clients as described in [Deployment](docs/DEPLOY.md). LAN HTTP transmits bearer tokens and data without encryption: restrict ingress, prevent Internet port forwarding, and use the server's literal RFC1918 private IPv4 address in the console. Binding all interfaces is not a LAN access-control rule. Verify SSH host keys when using SSH, and keep credentials out of shell history, source, Compose files and images. When authentication is enabled, rotate the bearer token by replacing its file and restarting the service; do not assume hot reload.

No security certification or completed RF hardware validation is claimed by these documents. Automated tests cover software behavior, not shielding performance or spectrum compliance.

## Optional authentication / 可选鉴权

If `GSMSNIFFER_TOKEN_FILE` is unset and `GSMSNIFFER_TOKEN` is unset or empty, both management listeners allow anonymous access. Every reachable client can read and delete observations, create and cancel jobs, and query operational state. This mode is for a trusted, restricted LAN only. Shielding requirements, redaction, explicit task acknowledgment and bounds still apply; they are not substitutes for access control.

配置文件路径后，文件不存在、不可读或为空会导致启动失败；非空无效 Token 同样导致启动失败，不会降级为匿名访问。启用鉴权时使用至少32字符的随机 Token；优先使用 `compose.auth.yml` 文件挂载，且每次重建均保留该覆盖文件。只用基础 Compose 且未设置环境 Token 会关闭鉴权。公开 `/api/v1/auth` 只报告是否要求登录，不返回密钥。
