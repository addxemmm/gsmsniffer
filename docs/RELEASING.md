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
- Manual runs default `publish=false`. Explicit `publish=true` or a matching version tag enables pushing only after validation.
- Published tags: `addxemmm/gsmsniffer:2.0.0` (demo runtime) and `:2.0.0-shielded` (optional dependencies). No implicit `latest` tag.
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
