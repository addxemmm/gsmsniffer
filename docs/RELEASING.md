# GitHub / Docker Hub 发布

## Current policy / 当前版本策略

Publish only **`addxemmm/gsmsniffer:2.1`**. The complete image includes gr-gsm/tshark and defaults to demo. Do not publish separate dependency, commit-qualified or `latest` tags. Source revision labels, provenance and the image digest provide traceability without additional tags.

只保留一个2.1镜像标签；demo与显式shielded使用同一镜像。模式、硬件权限与镜像版本是独立配置。端口为前端18083、后端8083，重启策略保持 `no`。

## Clean public source / 干净源码

The public repository already exists. Use its clean history for normal updates; do not reinitialize it or push the legacy history. For an independent reviewed source export:

```bash
python scripts/release_export.py --output release-export
```

Export is deny-by-default and rejects symlinks, binary captures, secrets/private-address patterns and forbidden paths; it creates a SHA-256 manifest. Manually review every exported file and its license/provenance. Never include old `.git`, workspace, logs, captures, images/PDFs or screenshots. The scanner does not read excluded private legacy files.

导出不是自动许可审查。先确认新增代码权利、NOTICE及外部包义务；未经核实的上游代码不复制、不重新标MIT。日常更新应使用已有的干净公开仓库。

## CI and publication / 自动化

Registry authentication and build/push actions follow the [official Docker workflow](https://docs.docker.com/build/ci/github-actions/push-multi-registries/).

- `ci.yml`: vet, tests, race tests, build, allowlist/tracked-file validation and a non-publishing full runtime image build.
- `release.yml`: manual dispatch only; validate VERSION, test/build and run RF-free demo smoke before publishing the single `2.1` tag. Creating a GitHub release/tag does not implicitly publish another image.
- Configure `DOCKERHUB_USERNAME` and `DOCKERHUB_IMAGE` for the confirmed namespace and repository. Keep `DOCKERHUB_TOKEN` only in GitHub secrets with the required repository scope; never commit credentials or use an account password.
- Manual validation is non-publishing by default. Set `publish=true` explicitly when publishing. The old `revision_tag` option is retired.
- Use the `dockerhub` environment and restrict workflow edits and release access. Publication does not SSH into servers, deploy containers or start RF jobs.

```bash
gh workflow run release.yml --ref main -f publish=true
```

Triggering the workflow does not itself confirm publication. Verify its successful result, source commit and actual Docker Hub digest. 触发成功不等于镜像发布成功；以实际构建、推送和拉取结果为准。

## Release checklist / 发布清单

1. VERSION, changelog, API docs, Postman, image labels and current examples agree on `2.1`.
2. Review the clean export and public diff; exclude private data and credentials.
3. Pass tests, full image build and demo-only smoke checks on both ports; do not imply RF acceptance.
4. Record source commit and final image digest. Review SBOM, vulnerability results and installed package copyright/source obligations before distribution.
5. Publish the single `2.1` tag and verify it can be pulled. Use no devices for the initial container check.
6. Separately deploy with the existing data volume, deliberately selected authentication configuration (retain `compose.auth.yml` for file-based login), and reviewed network bindings: loopback by default, or the explicit trusted-LAN configuration from [Deployment](DEPLOY.md). Confirm `RestartPolicy.Name=no`, both health endpoints and no active startup jobs. For LAN deployment, verify access from a second trusted client and preserve ingress restrictions; do not expose HTTP management ports to the Internet.
7. Only after successful deployment, inventory old tags and remove the explicitly approved obsolete tags. Preserve `2.1`, verify its digest before cleanup and check the final tag list. Delete by tag, not by a shared manifest digest. Keep rollback artifacts locally before cleanup; do not remove other projects' images or data volumes.
8. Publish sanitized release evidence. Retain the [2.0.0 historical record](releases/2.0.0.md), without presenting old tags or ports as current deployment guidance.

当前发布与服务器验收完成后，再清理历史标签。每次删除都限定本仓库与已核对的旧标签；`2.1`保持不动。历史文档中的旧digest是审计记录，不保证对应远程标签持续可拉取。

## Historical tag cleanup / 历史标签清理

`hub-cleanup.yml` is a separate manual workflow. Supply the verified `expected_digest` for `2.1`; `apply` defaults to `false` for inspection. After server acceptance, explicitly set `apply=true` to remove only the four known obsolete 2.0.0-series tags. The workflow checks the retained digest; it is not a wildcard repository purge. Its Docker Hub credential needs tag-deletion permission.

历史清理与发布分开执行，默认只检查。先核对2.1摘要与服务器验收，再明确启用删除；新版本及其他项目不在删除范围内。
