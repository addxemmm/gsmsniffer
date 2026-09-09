# 架构 / Architecture

```text
Browser / API client
       | UI + same-origin API :18083 / direct API-only :8083
       v
Go binary: embedded UI -> validation/auth -> job manager -> observations
                                      | opt-in shielded adapter
                                      v
                              gr-gsm / tshark processes
```

The management backend is Go 1.26 using the standard library; the frontend is embedded into the binary. No Node or Python web server is needed. RF executables remain independent upstream components: GNU Radio/gr-gsm can depend on Python. “Backend migrated to Go” does not mean “the complete image contains no Python”.

管理后端和 Web 服务为 Go；RF 工具链与管理面分离。外部程序的可用性与真实设备状态分别验收，安装成功不代表采集成功。

## Boundaries / 边界

- Default `demo` returns explicitly synthetic observations and never requires hardware.
- `shielded` must be selected explicitly. Jobs require acknowledgment and bounded duration; startup never starts a job.
- Token file takes precedence over the environment token. With no file configured and an unset/empty environment token, management routes allow anonymous access. Configured invalid credentials fail startup. A valid token enables bearer authentication on both listeners. `/healthz`, `/readyz` and `GET /api/v1/auth` are public; the auth endpoint reports only `data.required`.
- Identities are permanently masked; SMS is event-only and plaintext message bodies are not retained. Never use production subscriber data for screenshots, fixtures or issues.
- Both listeners default on: `GSMSNIFFER_ADDR=:18083` serves the embedded UI and same-origin API; `GSMSNIFFER_API_ADDR=:8083` is API-only (`/` returns 404). Both use the same authentication and manager. 双监听默认启用，共用鉴权和任务管理；前端保持同源调用，不跨域直连8083。
- Container has a writable data volume and temporary filesystem; root filesystem is read-only. No default host networking, privileged mode or USB forwarding.
- Clean publication starts from the allowlist, not the legacy workspace or existing Git history.

## State and retention / 状态与保留

The manager retains at most 100 jobs and 1,000 observations. With a data directory configured it writes bounded JSON state to `state.json` using a temporary file and rename; state loading rejects oversized/malformed content. Previously running jobs are marked failed after restart rather than restarted automatically. Each observation retains `source=demo|shielded|unknown`, independent of the current service mode. Identities/messages remain redacted when persisted and restored.

任务历史最多100条、观测最多1000条；重启不恢复运行中的射频任务。修改模式后仍须逐条看source，避免把旧demo结果当成屏蔽实验结果。数据卷不是数据库，未承诺数据库级事务或灾备；备份应在停止任务/服务后私下执行。清空观测不会等价于删除所有历史任务或私人备份。
