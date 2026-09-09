# REST API v1

Base URL: `http://127.0.0.1:8080/api/v1`. All API routes require `Authorization: Bearer TOKEN`, including the OpenAPI endpoint. Health routes `/healthz` and `/readyz` are unauthenticated and expose only liveness/readiness.

所有 API 均须 Bearer 鉴权；Token 不应放入 URL、日志或截图。独立监听器仅在设置 `GSMSNIFFER_API_ADDR` 后启用。

Standard JSON envelope (OpenAPI alone returns its native schema representation):

```json
{"code":"ok","message":"ok","data":{},"request_id":"request identifier"}
```

Use HTTP status as the primary success/error signal. Clients should preserve `request_id` when reporting problems; do not depend on human-readable message wording. 以 HTTP 状态判断结果，错误报告保留 request_id。

| Method | Path | Purpose |
|---|---|---|
| GET | `/status` | Service mode/status / 服务状态 |
| GET | `/capabilities` | Available adapters/dependencies / 能力 |
| GET | `/jobs` | List jobs / 任务列表 |
| POST | `/jobs` | Create bounded job / 创建任务 |
| GET | `/jobs/{id}` | Job detail / 任务详情 |
| DELETE | `/jobs/{id}` | Cancel/stop job / 取消任务 |
| GET | `/observations?kind=frequencies&limit=100&offset=0` | Paginated observations / 分页结果 |
| DELETE | `/observations` | Clear observations / 清空结果 |
| GET | `/openapi.json` | Machine-readable schema / OpenAPI |

Identities are permanently masked; SMS observations are redacted events, not plaintext message bodies. There is no unmask/export-raw API. 身份永久脱敏，短信只记录事件、不保留明文内容；这是相对旧工具的破坏性变更。

Each observation carries `source=demo|shielded|unknown`; distinguish per-record provenance when state survives a mode change. Do not treat old demo observations as hardware results. 每条记录来源独立于当前服务模式，unknown不可当作已确认RF数据。

Observation `kind`: `frequencies`, `imsi`, `sms`. Job `kind`: `scan`, `capture`. Band: `GSM900`, `DCS1800`. Capture mode: `imsi`, `sms`. Duration is positive and capped by `GSMSNIFFER_MAX_DURATION_SECONDS` (default 300). A scan accepts band only: omit frequency_mhz and mode. A capture requires mode and a valid in-band frequency_mhz. Pagination limit is 1–500 and offset is nonnegative. Job states are running, finished, cancelled and failed. Read the OpenAPI schema for exact field requirements and response shapes.

Example request for the default **demo** service (synthetic output only):

```bash
# Set TOKEN locally; do not save a real value in this document.
curl --fail-with-body http://127.0.0.1:8080/api/v1/jobs \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"kind":"scan","band":"GSM900","duration_seconds":60,"shielded_ack":true}'
```

The acknowledgment is an explicit operator assertion; it is not a measurement of shielding. The same request sent to a shielded service may request a real laboratory operation. Do not run it before completing the deployment checklist. 勾选确认不代表物理屏蔽已验证。

## Errors and client behavior / 错误处理

Expect 4xx for missing/invalid authentication, malformed JSON, invalid fields, missing job or conflicting state. Dependency failures should be treated as actionable errors, not empty successful captures. Handle non-JSON proxy errors, bounded request timeouts and retries carefully; do not blindly retry POST jobs because duplicate operations may result. Always query state before retrying.

Import `postman/gsmsniffer.postman_collection.json`; configure `baseUrl` and `token` locally. Collection defaults to read-only probes and an unauthorized-access test. Mutating demo examples are skipped unless allowMutations=true and the prior status request confirms demo mode. No real token is exported.
