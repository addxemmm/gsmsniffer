# Security / 安全

Use only owned test SIMs/terminals within an RF-shielded room or enclosure. Default demo mode, explicit job acknowledgment and duration limits complement physical isolation; none replaces it.

安全默认：API强制 Bearer、token文件优先、身份永久脱敏与短信仅事件、不自动启动任务、容器非root/只读/无特权、管理端口绑定loopback。不要把 Token、SSH 密码、IMSI、短信、RF捕获或旧Git历史上传至公开仓库。

Report suspected vulnerabilities privately to the repository owner using GitHub private vulnerability reporting when enabled. If it is unavailable, request a private contact without posting the exploit data or secrets publicly. Include affected version, environment, minimal synthetic reproduction, impact and request IDs. Never attach actual subscriber traffic.

If a secret is exposed, revoke/rotate it first, stop affected sessions, preserve a private incident record, then remove it from current content and history. A clean commit does not invalidate an exposed secret. Rebuild affected images and inspect image layers/artifacts where appropriate.

Before remote deployment: use an SSH tunnel/TLS management proxy, or explicitly enable HTTP only for trusted LAN clients as described in [Deployment](docs/DEPLOY.md). LAN HTTP transmits bearer tokens and data without encryption: restrict ingress, prevent Internet port forwarding, and use the server's literal RFC1918 private IPv4 address in the console. Binding all interfaces is not a LAN access-control rule. Verify SSH host keys when using SSH, and keep credentials out of shell history, source, Compose files and images. Rotate the bearer token by replacing its file and restarting the service; do not assume hot reload.

No security certification or completed RF hardware validation is claimed by these documents. Automated tests cover software behavior, not shielding performance or spectrum compliance.
