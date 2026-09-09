# 部署 / Deployment

## 1. Safe defaults / 默认配置

Default Compose enables both listeners and binds UI/same-origin API to host loopback `127.0.0.1:18083` and the API-only backend to `127.0.0.1:8083`, uses bridge networking, UID/GID 10001, a read-only root filesystem, `/tmp` tmpfs, a named data volume, dropped capabilities and `no-new-privileges`. Restart policy is `no`. No host networking, privileged mode, USB mapping or automatic job creation is present.

默认同时开放宿主loopback前端18083与后端8083，仍只部署demo管理台，不自动开始扫描/采集。前端保留同源API，独立后端不提供HTML界面。`docker compose down`保留数据卷；`down -v`会删除数据，须明确决定后操作。

| Variable | Default / behavior |
|---|---|
| `GSMSNIFFER_ADDR` | `:18083`, UI and same-origin API |
| `GSMSNIFFER_API_ADDR` | `:8083`, API-only backend; enabled by default / 默认启用 |
| `GSMSNIFFER_TOKEN_FILE` | Read token from this file; preferred over environment token |
| `GSMSNIFFER_TOKEN` | Fallback token; do not put in image or committed `.env` |
| `GSMSNIFFER_MODE` | `demo`; opt-in `shielded` |
| `GSMSNIFFER_DATA_DIR` | `/var/lib/gsmsniffer` |
| `GSMSNIFFER_MAX_DURATION_SECONDS` | `300` |
| `GSMSNIFFER_HEALTHCHECK_URL` | Unset: check both listeners; nonempty: override probe URL / 默认检查双端口，非空时覆盖探测URL |

Compose-only variables: `GSMSNIFFER_IMAGE`, `GSMSNIFFER_BIND`, `GSMSNIFFER_PORT` (host UI, default `18083`), `GSMSNIFFER_API_PORT` (host backend, default `8083`), `GSMSNIFFER_TOKEN_PATH`. 宿主前后端端口分别配置，容器监听地址用上表两个ADDR；两端口须不同。 Use an absolute host token path. `.env.example` contains placeholders only; pass `--env-file .env` explicitly if copying it.

## 2. Secrets / 密钥

The application requires a token at least 32 characters long. Generate at least 32 random bytes (`openssl rand -hex 32`). Save securely, then make the mounted file readable to UID 10001; on a root-managed Linux host use owner `10001:10001`, mode `0400`. Keep the enclosing `secrets` directory private. [Docker Compose file secrets](https://docs.docker.com/compose/how-tos/use-secrets/) are host file mounts, not encrypted secret storage; do not assume its `uid`/`mode` options will override host ownership. Rootless Docker/user namespaces may require adjusted host UID mapping.

若出现 permission denied，检查宿主Token所有者、容器映射和挂载路径，不要使用chmod 777。文件配置优先，错误/空文件应作为配置问题修复，而非把鉴权关闭。轮换Token后重启服务，未承诺热更新。

## 3. Build and run / 构建运行

Follow README for token setup, then:

```bash
export GSMSNIFFER_TOKEN_PATH="$(pwd)/secrets/api_token"
docker compose -f deploy/docker/compose.yml config --quiet
docker compose -f deploy/docker/compose.yml up -d --build
docker compose -f deploy/docker/compose.yml ps
docker compose -f deploy/docker/compose.yml logs --tail=100
```

Do not paste logs publicly before checking for sensitive data. `gsmsniffer healthcheck` probes both frontend and backend by default; set `GSMSNIFFER_HEALTHCHECK_URL` only for an intentional probe override. 双端口任一健康检查失败即视为失败；URL覆盖改变探测目标，不改变监听端口。 It checks HTTP liveness only; `/readyz` and authenticated capabilities complement it. None proves a connected SDR is usable.

The API listener is enabled by the base Compose file. `compose.api.yml` remains as an empty compatibility overlay for existing command lines; adding it no longer enables an otherwise-disabled port.

基础Compose已启用独立API；以下旧命令仍可用，`compose.api.yml`仅保留空兼容覆盖，不改变监听状态：

```bash
docker compose -f deploy/docker/compose.yml -f deploy/docker/compose.api.yml up -d --build
```

Both listeners require the same token; host UI 18083 and API 8083 stay loopback-only by default. The backend serves API/health only and returns 404 for `/`. 两端共用Token；后端根路径不提供前端页面。 Remote users should tunnel the management port using `ssh -L 18083:127.0.0.1:18083 USER@HOST`; verify the SSH host key independently. No private host or SSH password is stored in this project.

## 4. Shielded hardware integration / 屏蔽实验集成

Only owned test SIMs and terminals inside a verified shielded room/enclosure qualify. Confirm shielding before connecting RF paths. Building the optional target installs dependencies but grants **no device access** and does not activate RF jobs:

```bash
docker build --target shielded --build-arg VERSION=2.0.0 \
  -f deploy/docker/Dockerfile -t gsmsniffer:2.0.0-shielded .
```

An operator must separately review driver/device compatibility, exact USB device permissions, applicable group IDs and RF adapter support before selecting `GSMSNIFFER_MODE=shielded`. Prefer the narrowest device mapping practical for the device; never map all of `/dev` or enable privileged/host networking. Do not assume the demo Compose file is a finished hardware deployment recipe.

官方 Debian bookworm 提供 [gr-gsm](https://packages.debian.org/bookworm/gr-gsm) 和 [tshark](https://packages.debian.org/bookworm/tshark)。gr-gsm依赖GNU Radio/Python3；管理后端为Go，不代表可选镜像无Python。构建、依赖可用性、SDR设备兼容和真实RF采集是不同验收层级。

### Explicit integration override / 显式集成覆盖

`deploy/docker/compose.shielded.yml` is an opt-in **rootful Linux** integration template. It sets `user=0:0`, drops all capabilities then adds only `NET_RAW`, exposes `/dev/bus/usb` with character-device major 189 cgroup rules, and uses `HOME=/tmp`. This grants access to the USB bus, not just one device; review and narrow the mount/rules for the actual hardware. The default Compose command never loads it. It uses a separate `shielded_data` volume so root-written state does not break the non-root demo volume.

`tshark -p` disables promiscuous mode. `NET_ADMIN` is intentionally not granted without evidence it is necessary; actual dumpcap/kernel/device permission behavior still needs laboratory acceptance. The override preserves bridge networking, read-only root, no-new-privileges and no automatic jobs. No `privileged` is used. Stop demo first if reusing the same Compose project.

```bash
# Only AFTER physical shielding and owned-device checks; this starts the service, not an RF job.
docker compose -f deploy/docker/compose.yml -f deploy/docker/compose.shielded.yml config --quiet
docker compose -f deploy/docker/compose.yml -f deploy/docker/compose.shielded.yml up -d --build
```

该模板明确提升容器到受限root并暴露USB总线；不是默认部署，不是硬件权限已验证的证明。rootless或特殊用户命名空间环境需单独适配。切换回demo时用基础Compose重建，确保无USB映射和额外能力残留。

RF operation has not been validated merely by writing these files. Record the actual SDR model, driver versions, container digest, shielding checks, owned devices and bounded test results in private lab records; publish only sanitized summaries.

## 5. Upgrade, rollback and cleanup / 升级回滚

Record the current image digest and private data backup before upgrading. Pull a confirmed version tag or digest, stop the old container, recreate with the same reviewed configuration, and run demo smoke tests first. If acceptance fails, stop it and recreate with the previous digest. Do not run two instances against the same SDR.

Stop jobs before stopping the service. Clear observations via the authenticated API when intended, then stop the container. Container/volume removal and private backup deletion are separate actions. No deployment script publishes artifacts or starts jobs implicitly.
