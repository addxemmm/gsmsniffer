# Third-party components / 外部组件

The Go management layer uses the Go standard library. The single 2.1 container image additionally distributes Debian packages, including GNU Radio/gr-gsm and Wireshark/tshark and their dependencies. These retain their own licenses. No blanket MIT assertion covers the image.

| Component | Official source / package record | Review location |
|---|---|---|
| Go | https://go.dev/LICENSE | Go distribution LICENSE and bundled notices |
| Debian base | https://www.debian.org/legal/licenses/ | `/usr/share/doc/*/copyright` in the image |
| gr-gsm | https://packages.debian.org/bookworm/gr-gsm | `/usr/share/doc/gr-gsm/copyright` |
| tshark / Wireshark | https://packages.debian.org/bookworm/tshark | `/usr/share/doc/tshark/copyright` and wireshark-common notices |

Official Debian package records checked on 2026-09-09 confirm bookworm packages for gr-gsm and tshark. gr-gsm lists Python 3 and GNU Radio dependencies, so the complete image is not Python-free. Package availability does not prove a local Docker build or RF acceptance has succeeded.

发布前在最终镜像内生成实际包清单与SBOM，核对各包版权和对应源码提供义务。镜像 tag 可变：正式发布保存 digest、构建时间与来源修订号；不要把本表当作已完成法律审查或完整传递依赖清单。
