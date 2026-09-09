# GitHub / Docker Hub 发布

## Clean public source / 干净源码

Do not push the legacy repository/history. Deleting files in a new commit is insufficient. Export reviewed new files into an empty directory:

```bash
python scripts/release_export.py --output release-export
cd release-export
git init -b main
git add .
git diff --cached --stat
# Review every staged file and LICENSE/provenance before committing.
git commit -m 'Modernize GSM-SNIFFER 2.0.0'
# After confirming account/repository ownership:
# For a new namespace, create the public repository once:
# gh repo create OWNER/gsmsniffer --public --source=. --remote=origin --push
# The confirmed target already has an empty repository; attach it instead:
git remote add origin https://github.com/addxemmm/gsmsniffer.git
git push -u origin main
```

Export is deny-by-default and rejects symlinks, binary captures, secrets/private-address patterns and forbidden paths; it creates a SHA-256 manifest. A scanner is not proof of absence: manually review the export, including synthetic test fixtures, frontend assets, comments and notices. Never include old `.git`, workspace, logs, images/PDFs or screenshots. The scanner does not read excluded private legacy files.

导出不是自动许可审查。先确认新增代码权利、NOTICE及外部包义务；未经核实的上游代码不复制、不重新标MIT。

## CI and publication / 自动化

Registry authentication and build/push actions follow the [official Docker workflow](https://docs.docker.com/build/ci/github-actions/push-multi-registries/).

- `ci.yml`: vet, tests, race tests, build, allowlist/tracked-file validation and a non-publishing runtime Docker build.
- `release.yml`: manual `workflow_dispatch` or a `v*` tag. VERSION must match the selected tag. Validation/build and RF-free demo smoke are a separate job before publishing.
- Repository variable: `DOCKERHUB_USERNAME` (confirmed account or organization).
- Repository secret: `DOCKERHUB_TOKEN` (scoped Docker Hub access token). Do not commit its value or use an account password.
- Manual runs default `publish=false` and `revision_tag=false`. Explicit `publish=true` or a matching version tag enables pushing only after validation. For the current port-default revision, set both `publish=true` and `revision_tag=true`; do not use the existing version tags.
- Version-release tags remain `addxemmm/gsmsniffer:2.0.0` and `:2.0.0-shielded`; their historical digests must not be overwritten by the port-default revision. With `revision_tag=true`, tags are `VERSION-sha-SHA12` and `VERSION-sha-SHA12-shielded`, where `SHA12` is the first 12 characters of the workflow source commit (`GITHUB_SHA`). No implicit `latest` tag. 端口调整使用唯一revision标签，不把原2.0.0标签默认为新端口。
- Use the protected `dockerhub` environment for publisher approval if desired. Restrict who can create release tags and edit workflows.
- Publication never SSHs into servers, deploys containers or starts RF jobs.

CI and workflow configuration are not evidence that a public repo/image exists. Confirm the actual GitHub URL, Docker Hub tags and image digest after pushing. 首次目标账号未确认时保留占位符，禁止把配置文件写成“已发布”。

## Release checklist / 发布清单

1. VERSION, changelog, API docs, Postman and migration notes match.
2. Clean export reviewed; new repository history checked; no private data or credentials.
3. Tests and container smoke pass; record skipped RF acceptance honestly.
4. Pin/save source commit and final image digest. Review SBOM, vulnerability results and installed package copyright/source obligations before distribution.
5. Configure Docker Hub namespace/secret; execute workflow with explicit publishing intent.
6. Inspect both registry tags/digests. Pull and smoke-test the demo image without devices.
7. Publish sanitized release notes with exact checks and unresolved limitations. GitHub Release creation is a separate owner action, e.g. `gh release create v2.0.0 --title '2.0.0' --notes-file RELEASE_NOTES.md` after preparing reviewed notes.
8. Deploy only after a separate operator decision. Rollback uses the recorded prior digest, not mutable latest.

## Port-default revision without changing VERSION / 保持版本的端口调整

VERSION remains `2.0.0` for the current UI 18083 / backend 8083 source change. Preserve the historical `v2.0.0` release record and its original image digests. Do not republish the existing version tags to make them silently point at the revised defaults. If publishing this revision, use a new, unique revision-qualified tag and record the exact source commit and image digest; confirm workflow tag behavior before enabling push.

本次前端18083、后端8083调整暂不升版，不覆盖旧Release和digest。后续发布使用唯一revision镜像tag并记录源码提交与镜像摘要；未经核实不要让原有版本tag指向新构建。`docs/releases/2.0.0.md`保留当时真实端口，不能作为当前源码默认端口说明。

Dispatch the current revision explicitly / 显式发布本次修订：

```bash
gh workflow run release.yml --ref main -f publish=true -f revision_tag=true
```

After success, inspect the workflow source commit and registry digests. For VERSION `2.0.0`, use `addxemmm/gsmsniffer:2.0.0-sha-SHA12` (demo) or `:2.0.0-sha-SHA12-shielded`; replace `SHA12` with the verified 12-character commit prefix, not a guessed identifier. Triggering a workflow does not itself confirm publication.

运行成功后核对实际提交前12位与镜像digest，再部署对应revision标签；命令提交成功不等同于镜像已经发布。原 `2.0.0` / `2.0.0-shielded` 与旧验收记录继续保留。
