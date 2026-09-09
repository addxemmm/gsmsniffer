# GSM-SNIFFER 2.0.0 · Go laboratory console

> **使用范围：仅限自有测试 SIM、自有终端，在屏蔽室或屏蔽箱内进行隔离实验。不得接入或采集公众移动网络及第三方通信。射频任务前应确认屏蔽有效、测试设备归属及现场操作条件。**
>
> **Use only your own test SIMs and terminals inside an RF-shielded room or shielded enclosure. Do not connect to or collect public-network or third-party communications. Verify isolation and device ownership before any RF job.**

一个 Go 标准库管理后端，内嵌中英双语控制台与版本化 REST API。默认 `demo` 仅生成模拟数据、不访问 RF 硬件；容器启动不自动创建任务。`shielded` 是明确启用的实验模式，不代表已经完成硬件验收。

A Go standard-library management backend with an embedded bilingual console and versioned REST API. Default `demo` uses synthetic data, without RF hardware access. Startup creates no jobs. Explicit `shielded` mode is for laboratory integration; it is not a claim of hardware validation.

## 能力 / Features

- 同源 Web/API `:8080`；可选独立 API `:8083`，默认关闭。Same-origin UI/API, optional separate listener.
- 强制 Bearer Token；支持优先读取 token 文件。Mandatory bearer authentication, file-based secrets preferred.
- `/api/v1` 标准包络、OpenAPI、Postman 断言。Versioned envelopes, OpenAPI and Postman checks.
- 有界扫描/采集任务、取消与状态查询；身份永久脱敏，SMS只保留事件、不保留明文正文。Bounded jobs and cancellation; identities are permanently masked and SMS is event-only, without plaintext bodies.
- 多阶段 Docker：默认非 root、只读根文件系统、无特权及 USB 映射；RF 依赖单独 target。Hardened default image and opt-in RF dependencies.
- 新源码白名单导出；旧日志、捕获、图片/PDF、历史仓库不进入发布包。Allowlisted publication excludes legacy data and history.

## 快速开始 / Quick start

Requires Docker Engine + Compose v2 on the Linux server. From the new source checkout:

```bash
mkdir -p secrets
umask 077
openssl rand -hex 32 > secrets/api_token
# Container UID 10001 needs read access; do not make the parent secrets directory public.
# For a root-owned deployment directory, chown/chmod the token for the container:
sudo chown 10001:10001 secrets/api_token
sudo chmod 0400 secrets/api_token
export GSMSNIFFER_TOKEN_PATH="$(pwd)/secrets/api_token"
docker compose -f deploy/docker/compose.yml up -d --build
curl --fail http://127.0.0.1:8080/healthz
```

Open `http://127.0.0.1:8080`; use the generated token in the console. Remote access should use an SSH tunnel (`ssh -L 8080:127.0.0.1:8080 USER@HOST`) or an authenticated TLS reverse proxy. Do not publish management ports directly to the Internet. The sample contains no server passwords or account tokens.

打开本机控制台，输入生成的 Token。远程使用 SSH 隧道或带 TLS 的管理代理，不直接暴露管理端口。示例不包含服务器凭据。Token 文件权限、清理和回滚见 [部署指南](docs/DEPLOY.md)。

```bash
# Development: Go 1.26; no Node build step required.
export GSMSNIFFER_TOKEN_FILE=/absolute/path/to/api_token
export GSMSNIFFER_ADDR=127.0.0.1:8080
go test ./...
go run ./cmd/server
```

## 文档 / Documentation

[文档索引](docs/README.md) · [快速开始](docs/QUICKSTART.md) · [架构](docs/ARCHITECTURE.md) · [API](docs/API.md) · [Web UI](docs/WEB_UI.md) · [部署](docs/DEPLOY.md) · [迁移](docs/MIGRATION.md) · [测试](docs/TESTING.md) · [发布](docs/RELEASING.md) · [安全](SECURITY.md)

## 发布与许可 / Publication and licensing

Version `2.0.0` introduces breaking changes from the legacy tool. Repository creation, image publication and actual deployment are separate operations: workflow files alone do not prove that an image has been pushed. Docker Hub examples use `addxemmm/gsmsniffer:2.0.0`; The target namespace is confirmed; verify the tag exists before pulling.

`2.0.0` 是破坏性改造版本，不保留旧工具的明文IMSI/SMS展示或导出兼容。No raw IMSI/SMS display or export compatibility is retained. 发布流程不自动登录实验服务器，不自动启动射频任务。旧工作目录与 Git 历史不得直接推入新公开项目。新增部分和外部运行组件的许可边界见 [NOTICE](NOTICE.md) 与 [THIRD_PARTY_NOTICES](THIRD_PARTY_NOTICES.md)；不将未核实许可的旧代码重新声明为 MIT。
