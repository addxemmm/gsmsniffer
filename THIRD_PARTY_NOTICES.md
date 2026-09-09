# Third-party components / 外部组件

The Go management layer uses the Go standard library. The single 2.1 container image distributes Ubuntu Jammy packages (UHD 4.1, GNU Radio, gr-osmosdr, Wireshark/tshark and dependencies) plus gr-gsm built from a fixed Debian upstream source archive. These retain their own licenses. No blanket MIT assertion covers the image.

| Component | Official source / package record | Review location |
|---|---|---|
| Go | https://go.dev/LICENSE | Go distribution LICENSE and bundled notices |
| Ubuntu base and packages | https://ubuntu.com/legal/terms-and-policies/open-source | `/usr/share/doc/*/copyright` in the image |
| gr-gsm | https://deb.debian.org/debian/pool/main/g/gr-gsm/gr-gsm_1.0.0~20220727.orig.tar.xz | `/usr/share/doc/gr-gsm/COPYING` and complete upstream `source.tar.xz` |
| UHD | https://packages.ubuntu.com/jammy/uhd-host | `/usr/share/doc/uhd-host/copyright` and libuhd notices |
| tshark / Wireshark | https://packages.ubuntu.com/jammy/tshark | `/usr/share/doc/tshark/copyright` and wireshark-common notices |

The gr-gsm upstream archive is pinned to SHA256 `968f262dd7090b1d5951a5c743296c8ba063cc71127667dacb18aa4bcdb22153` and is built without source modifications; the repository Dockerfile contains its build instructions. It targets GNU Radio 3.10. Jammy provides the UHD 4.1 ABI matching the reference LTE/GSM installations, but not a gr-gsm binary package. This image still includes Python. Build/package checks do not establish hardware reception.

BlackSDR vendor FPGA/FX3 binaries are not bundled. Operators mount their own validated, matching firmware directory read-only. A firmware hash proves file identity, not redistribution rights. Do not copy vendor blobs into the public source or published image without separately established rights.

发布前在最终镜像内生成实际包清单与SBOM，核对各包版权和对应源码提供义务。镜像 tag 可变：正式发布保存 digest、构建时间与来源修订号；不要把本表当作已完成法律审查或完整传递依赖清单。
