# 当前发布状态与验证证据

[功能矩阵](capabilities.md) · [统一版本规则](version-policy.md) · [构建与发布](build-release.md)

核对日期：2026-10-09。目标发布：**v0.11.1**。

## 代码基线

2026-10-09 检查时，远程 `origin/master=62236c0`，对应正式 `v0.11.0`；旧本地 `master=4390bb2` 仍停在 v0.10.3 且有未提交修改，因此本轮以远程 v0.11.0 为发布基线。版本首次引入详情见 [功能矩阵](capabilities.md)。

| 范围 | 当前状态 | 边界 |
| --- | --- | --- |
| Wails Runtime/Tray/AutoStart | 已在正式基线 | 需消费者显式调用 |
| Theme / UI / About / 配置 / Secure Config | 已在正式基线 | 不代表消费者界面或配置已迁移 |
| Windows NSIS + Linux raw/deb/tar.gz + SHA256 | 已在正式基线 | 发布仍需运行 CI |
| Linux .deb 的 systemd 自动安装/启停 | v0.10.1 已发布、v0.11.0 已包含 | 仅 opt-in 的服务程序 |
| Updater：Check/Download/Verify/Windows Install+Restart | v0.11.0 已发布 | Portable 不自覆盖，machine-scope 手动安装 |
| Linux CLI systemd 服务管理 | v0.11.1 合入目标 | 业务 CLI 必须主动注册，Kit 不会自动添加命令 |

## 已有历史验证

| 证据 | 验证内容 |
| --- | --- |
| [Kit Installer CI 36323530206](https://github.com/wanstu/wails-desktop-kit/actions/runs/36323530206) | v0.10.0 Windows 安装/覆盖/卸载、三平台基础测试与 workflow E2E |
| [Updater CI 36421849097](https://github.com/wanstu/wails-desktop-kit/actions/runs/36421849097) | Windows 已安装实例 Install/Restart、Portable 拒绝自覆盖 |
| [Know Me 消费者 CI 36326162758](https://github.com/wanstu/know_me/actions/runs/36326162758) | 真实消费者 Windows Setup 安装/卸载和数据保留 |

历史 CI 不能证明新 tag 的 CI 已通过。发布 v0.11.1 后必须补充该 tag 的测试/发布链接，并将“发布完成”与“消费者已升级”分别确认。

## 后续工作

优先审查 ADM 和 Know Me 重复的更新状态机、Know Me 和 MCP Center 的 Linux 服务 CLI、跨项目版本诊断。抽取通用边界前先保证实际两个消费者可以使用，且不破坏既有启动、退出、权限及安装行为。MCP/OAuth 协议保持 mcp-center-core 独立，不纳入 Kit。
