# 部署 / Deployment

## 1. Safe defaults / 默认配置

Default Compose enables both listeners and binds UI/same-origin API to host loopback `127.0.0.1:18083` and the API-only backend to `127.0.0.1:8083`, uses bridge networking, UID/GID 10001, a read-only root filesystem, `/tmp` tmpfs, a named data volume, dropped capabilities and `no-new-privileges`. Restart policy is `no`. No host networking, privileged mode, USB mapping or automatic job creation is present.

默认同时开放宿主loopback前端18083与后端8083，仍只部署demo管理台，不自动开始扫描/采集。前端保留同源API，独立后端不提供HTML界面。`docker compose down`保留数据卷；`down -v`会删除数据，须明确决定后操作。

| Variable | Default / behavior |
|---|---|
| `GSMSNIFFER_ADDR` | `:18083`, UI and same-origin API |
| `GSMSNIFFER_API_ADDR` | `:8083`, API-only backend; enabled by default / 默认启用 |
| `GSMSNIFFER_TOKEN_FILE` | Optional token file; takes precedence. Configured missing/unreadable/empty file fails startup |
| `GSMSNIFFER_TOKEN` | Unset/empty with no token file: anonymous access. Valid nonempty token: login required. Invalid nonempty token: startup failure |
| `GSMSNIFFER_MODE` | `demo`; opt-in `shielded` |
| `GSMSNIFFER_DATA_DIR` | `/var/lib/gsmsniffer` |
| `GSMSNIFFER_MAX_DURATION_SECONDS` | `300` |
| `GSMSNIFFER_HEALTHCHECK_URL` | Unset: check both listeners; nonempty: override probe URL / 默认检查双端口，非空时覆盖探测URL |

Compose-only variables: `GSMSNIFFER_IMAGE`, `GSMSNIFFER_BIND` (host UI bind, default `127.0.0.1`), `GSMSNIFFER_API_BIND` (host backend bind, default `127.0.0.1`), `GSMSNIFFER_PORT` (host UI, default `18083`), `GSMSNIFFER_API_PORT` (host backend, default `8083`), `GSMSNIFFER_TOKEN_PATH`. 宿主前后端地址和端口分别配置，容器监听地址用上表两个ADDR；两端口须不同。 Use an absolute host token path only with the optional `compose.auth.yml` overlay; the base file requires no secret. `.env.example` contains placeholders only; pass `--env-file .env` explicitly if copying it.

## 2. Optional authentication / 可选鉴权

With `GSMSNIFFER_TOKEN_FILE` unset and `GSMSNIFFER_TOKEN` unset or empty, both ports allow anonymous access and the console connects automatically. **Every client able to reach either management port can read/delete observations and create/cancel jobs.** Use this only on a trusted restricted LAN. Public `GET /api/v1/auth` reports `data.required` without exposing any secret.

未配置 Token 即免登录，不需要生成或挂载密钥文件。配置有效 Token 后两端口共用 Bearer 鉴权。文件路径一旦配置，文件缺失、不可读、内容为空，或 Token 非空但不满足长度要求，均导致启动失败，不会静默降级为匿名模式。基础 Compose 不强制 secret；只设置 `GSMSNIFFER_TOKEN_PATH` 不会启用文件鉴权，必须同时加载下述覆盖文件。

To require login, use at least 32 characters of random token. An environment `GSMSNIFFER_TOKEN` is supported, but a file-secret overlay is preferred:

```bash
mkdir -p secrets
chmod 0700 secrets
umask 077
openssl rand -hex 32 > secrets/api_token
# Save the token privately before restricting its ownership to container UID 10001.
sudo chown 10001:10001 secrets/api_token
sudo chmod 0400 secrets/api_token
export GSMSNIFFER_TOKEN_PATH="$(pwd)/secrets/api_token"
docker compose -f deploy/docker/compose.yml -f deploy/docker/compose.auth.yml config --quiet
docker compose -f deploy/docker/compose.yml -f deploy/docker/compose.auth.yml up -d --build
```

Always retain `-f deploy/docker/compose.auth.yml` when recreating a file-authenticated service. To intentionally switch to anonymous access, omit that overlay, unset/empty `GSMSNIFFER_TOKEN`, and recreate the same project, retaining the data volume. Old token files can remain privately stored for rollback; their presence on disk alone does not enable authentication. Add the auth overlay alongside `compose.shielded.yml` if that explicitly reviewed deployment needs login.

[Docker Compose file secrets](https://docs.docker.com/compose/how-tos/use-secrets/) are host file mounts, not encrypted secret storage; do not assume its `uid`/`mode` options override host ownership. Rootless Docker/user namespaces may require adjusted host UID mapping. 若出现 permission denied，检查所有者、容器映射和路径，不要使用 chmod 777。轮换 Token 后重启服务，未承诺热更新。不要将密钥写入镜像、公开文件或日志。

## 3. Build and run / 构建运行

For the default no-token deployment (add the auth overlay above if login is required):

```bash
export GSMSNIFFER_TOKEN='' # Also overrides any token in a Compose .env file.
unset GSMSNIFFER_TOKEN_FILE
docker compose -f deploy/docker/compose.yml config --quiet
docker compose -f deploy/docker/compose.yml up -d --build
docker compose -f deploy/docker/compose.yml ps
docker compose -f deploy/docker/compose.yml logs --tail=100
```

Do not paste logs publicly before checking for sensitive data. `gsmsniffer healthcheck` probes both frontend and backend by default; set `GSMSNIFFER_HEALTHCHECK_URL` only for an intentional probe override. 双端口任一健康检查失败即视为失败；URL覆盖改变探测目标，不改变监听端口。 It checks HTTP liveness only; `/readyz` and capabilities complement it (provide a token when authentication is enabled). None proves a connected SDR is usable.

The API listener is enabled by the base Compose file. `compose.api.yml` remains as an empty compatibility overlay for existing command lines; adding it no longer enables an otherwise-disabled port.

基础Compose已启用独立API；以下旧命令仍可用，`compose.api.yml`仅保留空兼容覆盖，不改变监听状态：

```bash
docker compose -f deploy/docker/compose.yml -f deploy/docker/compose.api.yml up -d --build
```

Both listeners share the selected authentication mode; host UI 18083 and API 8083 stay loopback-only by default. The backend serves API/health only and returns 404 for `/`. 两端共用可选鉴权配置；后端根路径不提供前端页面。 An SSH tunnel remains an option: `ssh -L 18083:127.0.0.1:18083 USER@HOST`; verify the SSH host key independently. No private host or SSH password is stored in this project.

### Trusted LAN HTTP / 受信任局域网 HTTP

无需域名或 HTTPS 证书时，可显式允许局域网访问。保留已选择的鉴权配置、数据卷、镜像 `addxemmm/gsmsniffer:2.1` 和 `restart: "no"`，仅修改部署环境文件中的宿主绑定地址：

```dotenv
GSMSNIFFER_BIND=0.0.0.0
GSMSNIFFER_API_BIND=0.0.0.0
GSMSNIFFER_PORT=18083
GSMSNIFFER_API_PORT=8083
```

```bash
# Keep the existing project name and env file; do not create a second deployment.
# For file authentication, retain -f deploy/docker/compose.auth.yml in both commands.
docker compose --env-file /absolute/path/to/deployment.env -f deploy/docker/compose.yml config --quiet
docker compose --env-file /absolute/path/to/deployment.env -f deploy/docker/compose.yml up -d --no-build
# Run from another trusted LAN client; HOST is the server's RFC1918 private IPv4 address.
curl --fail http://HOST:18083/healthz
curl --fail http://HOST:8083/healthz
```

浏览器访问 `http://HOST:18083`；直接 API 基址为 `http://HOST:8083/api/v1`。前端通过 18083 同源接口登录和操作，不需要跨域访问 8083。使用既有 Compose 项目名（如原命令有 `-p`，继续带上）和同一个环境文件，以保留原数据卷。后续更新仍应传入该环境文件，避免意外恢复 loopback。

The console permits HTTP token submission only on loopback or literal RFC1918 private IPv4 hosts; HTTPS is also supported. A LAN hostname is not automatically trusted: use the private IPv4 address, or configure HTTPS. **HTTP does not encrypt bearer tokens or data.** Only use this option on a trusted LAN; prevent Internet port forwarding and restrict ingress to intended clients using appropriate host/network firewall rules. Binding `0.0.0.0` listens on every IPv4 interface, not just the LAN interface; it is not an access-control rule. Verify access from a second LAN machine and verify that untrusted networks cannot reach either port. If authentication is enabled, the API requires its bearer token; otherwise any reachable client can manage jobs and data; the browser transport check is not a substitute for firewall protection.

**HTTP 明文传输 Token 与数据。** `0.0.0.0` 表示所有 IPv4 网卡，不会自动限制为局域网；须结合宿主及网络边界规则限制可信来源，禁止公网端口转发。浏览器的地址检查不代替防火墙，配置 Token 时直接 API 调用须携带 Token；未配置时所有可达客户端均可匿名管理任务与数据。扩大网络监听不会启用 RF 设备、创建任务或改变不开机自启策略。

## 4. Shielded hardware integration / 屏蔽实验集成

Only owned test SIMs and terminals inside a verified shielded room/enclosure qualify. Confirm shielding before connecting RF paths. The single `addxemmm/gsmsniffer:2.1` image already includes gr-gsm/tshark. Installing these dependencies grants **no device access** and does not activate RF jobs. The same image is used for demo and shielded integration; no separate image tag or build target is required.

2.1 使用一个完整依赖镜像。默认 demo、无 USB 映射，启用依赖不等于启用硬件权限或开始任务。

An operator must separately review driver/device compatibility, exact USB device permissions, applicable group IDs and RF adapter support before selecting `GSMSNIFFER_MODE=shielded`. Prefer the narrowest device mapping practical for the device; never map all of `/dev` or enable privileged/host networking. Do not assume the demo Compose file is a finished hardware deployment recipe.

官方 Debian bookworm 提供 [gr-gsm](https://packages.debian.org/bookworm/gr-gsm) 和 [tshark](https://packages.debian.org/bookworm/tshark)。gr-gsm依赖GNU Radio/Python3；管理后端为Go，不代表完整镜像无Python。构建、依赖可用性、SDR设备兼容和真实RF采集是不同验收层级。

### Explicit integration override / 显式集成覆盖

`deploy/docker/compose.shielded.yml` is an opt-in **rootful Linux** integration template. It sets `user=0:0`, drops all capabilities then adds only `NET_RAW`, exposes `/dev/bus/usb` with character-device major 189 cgroup rules, and uses `HOME=/tmp`. This grants access to the USB bus, not just one device; review and narrow the mount/rules for the actual hardware. The default Compose command never loads it. It changes runtime mode, device permissions and state isolation, not the image version. It uses a separate `shielded_data` volume so root-written state does not break the non-root demo volume.

`tshark -p` disables promiscuous mode. `NET_ADMIN` is intentionally not granted without evidence it is necessary; actual dumpcap/kernel/device permission behavior still needs laboratory acceptance. The override preserves bridge networking, read-only root, no-new-privileges and no automatic jobs. No `privileged` is used. Stop demo first if reusing the same Compose project.

```bash
# Only AFTER physical shielding and owned-device checks; this starts the service, not an RF job.
docker compose -f deploy/docker/compose.yml -f deploy/docker/compose.shielded.yml config --quiet
docker compose -f deploy/docker/compose.yml -f deploy/docker/compose.shielded.yml up -d --build
```

该模板明确提升容器到受限root并暴露USB总线；不是默认部署，不是硬件权限已验证的证明。rootless或特殊用户命名空间环境需单独适配。切换回demo时用基础Compose重建，确保无USB映射和额外能力残留。

RF operation has not been validated merely by writing these files. Record the actual SDR model, driver versions, container digest, shielding checks, owned devices and bounded test results in private lab records; publish only sanitized summaries.

## 5. Upgrade, rollback and cleanup / 升级回滚

Record the current image digest and private data backup before upgrading. Current deployments use `addxemmm/gsmsniffer:2.1`; preserve a local rollback image before old registry tags are removed. Pull a confirmed version tag or digest, stop the old container, recreate with the same reviewed configuration, and run demo smoke tests first. If acceptance fails, stop it and recreate with the previous digest. Do not run two instances against the same SDR.

Stop jobs before stopping the service. Clear observations via the management API when intended (authenticate if enabled), then stop the container. Container/volume removal and private backup deletion are separate actions. No deployment script publishes artifacts or starts jobs implicitly.
