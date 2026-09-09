# Changelog

- 2.1 workflow correction: scan frequencies first, select a detected channel, then explicitly scan IMSI/SMS. Unified workspace with live frequency/result tables, source validation, selection invalidation and task stop.

## 2.1 — 2026-09-09

- Make authentication optional: an unset/empty environment token with no token file allows anonymous console auto-connect; configured valid credentials require login, while malformed configured credentials fail startup. Add public `/api/v1/auth` mode discovery and an opt-in file-secret Compose overlay.

- Publish one complete image: `addxemmm/gsmsniffer:2.1`, including gr-gsm/tshark; no separate dependency or commit-qualified image tags.
- Keep demo as the default, without device access or automatic jobs; the shielded Compose override selects mode and hardware permissions only.
- Standardize frontend `18083` and API-only backend `8083`; retain loopback host bindings and `restart: "no"` (no boot startup).
- Align version metadata, deployment examples and release workflow with the single-tag policy. Preserve 2.0.0 as a historical record.
- Physical RF acceptance remains separate; see [2.1 notes](docs/releases/2.1.md).

## 2.0.0 — 2026-09-09

### Breaking / 破坏性变更
- Replace the management backend with Go and an embedded bilingual console.
- Permanently mask identities and retain SMS events only, without plaintext bodies; legacy raw IMSI/SMS display/export is not retained.
- Introduce authenticated `/api/v1`, standard response envelopes and OpenAPI.
- Default to synthetic demo mode; RF operation is explicit and requires owned test devices in shielded facilities.
- Exclude legacy logs, captures, images/PDFs and Git history from the public release.

### Added / 新增
- Bounded job lifecycle, cancellation and redacted observations.
- Hardened Docker defaults and a separate optional RF dependency target.
- CI validation, deliberate Docker Hub publishing, allowlisted export and release scanning.
- API, Postman, deployment, architecture, migration, security and testing documentation.

### Validation / 验证
- Record actual software/container results in release notes.
- Shielded RF hardware acceptance remains a separate requirement; this changelog does not claim it has passed.
- This entry describes source changes, not confirmation of public repository/image publication.
