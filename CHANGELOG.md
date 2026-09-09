# Changelog

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
