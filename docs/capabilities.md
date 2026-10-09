# Kit 功能清单与消费者核对

> 维护原则：正式 tag 的源代码决定“已发布”；主线未发布提交决定“待发布”；业务工程的 go.mod / workflow 决定“是否接入”。文档不能替代这三项证据。

## v0.11.2 合并目标功能

| 模块 | Kit API / 路径 | 正式版本首次引入 | v0.11.2 目标状态与边界 |
| --- | --- | --- | --- |
| Desktop Runtime | 根包 Config / Hooks / Controller | v0.2.0 | 已包含；Wails v2 生命周期、窗口、托盘、单实例及退出策略 |
| Login AutoStart | autostart | v0.2.x | 已包含；Windows / Linux / macOS 登录自启，不等于 Linux systemd 服务 |
| CSS / UI / About | ui、buildmeta | v0.2.x / v0.9.0 | 已包含基础组件、设计 token、About 资源；不代表每个消费者页面已统一 |
| Theme | theme | v0.5.0 | 已包含远程主题、内置 fallback、缓存与校验 |
| 路径与配置 | paths / atomicfile / jsonstore | v0.4.0 / v0.8.0 | 已包含；应用 schema 与数据迁移仍由消费者负责 |
| 秘密配置 | secureconfig | v0.4.0 | 已包含加密配置；不自动迁移业务明文配置 |
| 图标与构建信息 | icon / desktopkit buildmeta | v0.3.1 / v0.9.0 | 已包含；消费者须在 build workflow 调用 |
| Linux / Windows 打包 | packaging / desktopkit package | v0.6.0 / v0.10.0 | 已包含 Linux raw/deb/tar.gz、Windows Setup 和 SHA256 |
| Linux systemd 安装 | packaging/linux_systemd | v0.10.1 | 已包含；仅显式配置 --systemd 的应用启用 |
| Linux 服务持久化配置 | packaging/linux_systemd | v0.10.2（维护线） | 已合入 EnvironmentFile/Environment，应用必须显式设置 |
| Debian 元数据 | packaging/linux_metadata / linux_appstream | v0.10.2 / v0.10.3（维护线） | 已合入 Installed-Size、版权文档和 AppStream 组件 ID 命名 |
| Debian tar 父目录 | packaging/linux | v0.11.2 | 增加显式目录条目和覆盖桌面资源/systemd/AppStream 的测试 |
| Updater Core | updater | v0.11.0 | 已包含 Check/Download/Verify 与 Windows user-scope Install/Restart；Portable 不自覆盖，machine-scope 不自动安装 |
| Linux 服务管理 CLI | servicecontrol | v0.11.1 | 已包含 status/start/stop/restart/enable/disable；业务须显式注册命令，非安装工具 |

## 2026-10-09 消费者源码快照

| 消费者 | 检查到的 Kit Go Module 依赖 | 现有代码 | 不可误判为 |
| --- | --- | --- | --- |
| AI Dev Manager (开发 worktree) | v0.11.0 | 有独立 Desktop UpdateManager，调用 kit/updater；部分共享能力接入 | 不能称整个 ADM 已全面 Kit 化；未提交 updater 修改不算发布 |
| Know Me (开发 worktree) | v0.11.0 | 使用 Runtime、paths/jsonstore、theme/ui 与 updater；Linux CLI 正在开发 | 不能称 v0.11.1 已被正式消费者采用 |
| MCP Center (master 工作树) | v0.10.0 | 服务 CLI 控制、端口修改、doctor 等业务实现；已使用可复用 core | 不能称这些命令来自 Kit v0.11.1；当前存在未提交改动 |
| SSH Client (开发 worktree) | v0.10.0 | 使用 Runtime、autostart、secureconfig、jsonstore、theme | 不能称已经接入 updater 或新 servicecontrol |

这里记录的是检查时的源码状态，而不是线上发布版本。Go 依赖与 GitHub reusable workflow 是**两项独立版本引用**，分别检查。

## 可以进一步抽象，但不包含在 v0.11.2

| 来源 | 候选公共模块 | 决策 |
| --- | --- | --- |
| ADM 与 Know Me Desktop UpdateManager | updater/controller（任务状态、并发互斥、进度、取消、错误呈现；由产品提供回调） | 优先研究；首先对齐 API，不搬业务 UI |
| MCP Center / Know Me Linux CLI | servicecontrol 的 CLI 命令注册和系统服务诊断，可增加配置驱动模板 | 已有低层 servicecontrol；扩展必须有 2 个消费者测试 |
| ADM / Know Me / MCP Center | buildinfo / 版本输出 / doctor 基础结构 | 可以统一元数据和诊断约定，不抽取业务检查项 |
| ADM / Know Me / SSH | config/schema migrator 与安全配置 | 先复用现有 paths/jsonstore/secureconfig，避免平行实现 |
| ADM / Know Me Web | 首访注册、后台登录/会话/权限 | 不放进 Wails Desktop Kit；若确需跨项目共享，应独立 Go Web 组件 |
| MCP Center Core | MCP 协议、OAuth、路由代理 | 继续留在 mcp-center-core；不要挪进 Kit |

## 接入验收

每个消费者至少分别验证：`go.mod` 版本、workflow tag 与 helper 版本、构建产物、特性实际调用、退出与更新语义。单独包含依赖不等于已使用功能；已有 Kit CSS 不等于整站 UI 已统一。
