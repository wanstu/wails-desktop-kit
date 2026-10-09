# 当前发布状态与验证证据

[功能矩阵](capabilities.md) · [统一版本规则](version-policy.md) · [构建与发布](build-release.md)

核对日期：2026-10-09。当前正式基线 **v0.11.2**；下一补丁版本目标 **v0.11.3**，完成脚本版本检测与 systemd 生命周期真实验收。

## 代码基线

v0.11.1 正式基线为 `b2513b9`，远程 `origin/master` 已包含 Windows 安装与更新以及 Linux `servicecontrol`，但遗漏 v0.10.2/v0.10.3 维护线两次提交。v0.11.2 开发分支基于正式 v0.11.1，逐项合入维护线功能并补充真实 Debian 包 CI；通过有独立父提交的合并提交使 `v0.10.3` 成为新分支祖先。旧本地 `master=4390bb2` 的未提交修改不覆盖、不清理。发布证据以新 tag 和 CI 为准。

| 范围 | 当前状态 | 边界 |
| --- | --- | --- |
| Wails Runtime/Tray/AutoStart | 已在正式基线 | 需消费者显式调用 |
| Theme / UI / About / 配置 / Secure Config | 已在正式基线 | 不代表消费者界面或配置已迁移 |
| Windows NSIS + Linux raw/deb/tar.gz + SHA256 | 已在正式基线 | 发布仍需运行 CI |
| Linux .deb 的 systemd 自动安装/启停 | v0.10.1 已发布、v0.11.0 已包含 | 仅 opt-in 的服务程序 |
| Updater：Check/Download/Verify/Windows Install+Restart | v0.11.0 已发布 | Portable 不自覆盖，machine-scope 手动安装 |
| Linux CLI systemd 服务管理 | v0.11.1 已发布 | 业务 CLI 必须主动注册，Kit 不会自动添加命令 |
| Debian 持久化配置和 metadata | v0.11.2 合入目标 | 补齐 v0.10.2 的服务配置、Installed-Size 和版权文件 |
| AppStream 组件 ID 命名 | v0.11.2 合入目标 | 补齐 v0.10.3 修复；保留原 AppStream ID 校验 |
| Debian tar 父目录与安装包验证 | v0.11.2 合入目标 | tar 父目录显式写入，单测及 Linux CI 真实 .deb 解包验证 |

## 已有历史验证

| 证据 | 验证内容 |
| --- | --- |
| [Kit Installer CI 36323530206](https://github.com/wanstu/wails-desktop-kit/actions/runs/36323530206) | v0.10.0 Windows 安装/覆盖/卸载、三平台基础测试与 workflow E2E |
| [Updater CI 36421849097](https://github.com/wanstu/wails-desktop-kit/actions/runs/36421849097) | Windows 已安装实例 Install/Restart、Portable 拒绝自覆盖 |
| [Know Me 消费者 CI 36326162758](https://github.com/wanstu/know_me/actions/runs/36326162758) | 真实消费者 Windows Setup 安装/卸载和数据保留 |

v0.11.1 跨平台 CI：[37886788530](https://github.com/wanstu/wails-desktop-kit/actions/runs/37886788530)。v0.11.2 主线跨平台与 Windows 安装器工作流 CI：[37894619736](https://github.com/wanstu/wails-desktop-kit/actions/runs/37894619736)，均已通过。新版本须以自己的主线 CI 再次验收。

## Kit 版本治理补充验收（v0.11.2 之后开发）

- `doctor/upgrade`：覆盖消费者构建脚本中 `desktopkit@v...` 固定引用及 `desktopkit-cli-version`，并支持 `go.mod` 单行和多行 require。动态变量不被擅自改写。
- Linux：新增在真实 systemd 宿主上对 Kit 生成的 `.deb` 执行 `dpkg --install`、升级、服务重启、`--remove`、`--purge`，验证 `/var/lib/<package>` 文件在全流程保留。CI 必须真实运行，不具备 systemd 时直接失败而不是跳过。

PR 及本地测试已建立，Linux systemd 全流程实机测试已通过：[Kit CI 37898065585](https://github.com/wanstu/wails-desktop-kit/actions/runs/37898065585)。必须在 v0.11.3 的最终主线 CI 通过后才能宣告发布完成。业务应用的升级和集成不属于本次工作。

## 后续工作

优先审查 ADM 和 Know Me 重复的更新状态机、Know Me 和 MCP Center 的 Linux 服务 CLI、跨项目版本诊断。抽取通用边界前先保证实际两个消费者可以使用，且不破坏既有启动、退出、权限及安装行为。MCP/OAuth 协议保持 mcp-center-core 独立，不纳入 Kit。
