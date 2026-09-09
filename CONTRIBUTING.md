# Contributing / 贡献指南

Use synthetic fixtures only. Keep every RF experiment within owned test SIM/device scope and a shielded room/enclosure. Never commit credentials, raw captures, identity records or message content.

1. Use Go 1.26; run `gofmt`, `go vet ./...`, `go test ./...` and `go test -race ./...` on a supported host.
2. Update OpenAPI, Postman checks and bilingual docs when changing an endpoint.
3. Add tests for validation, authentication, cancellation, redaction and error paths.
4. Preserve default demo mode, mandatory authentication and no startup job. Shell command interpolation must not be introduced for request parameters.
5. Run `python scripts/release_export.py --output release-export` and review the resulting manifest before publication.
6. Describe what was actually tested, separately from tests needing RF hardware.

新增依赖先记录来源和许可，不复制未核实许可的旧workspace代码。版本与CHANGELOG同步；破坏性变更须写入迁移文档。PR中不放私有服务器地址和真实账号资料。
