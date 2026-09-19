# 进度、证据与待办

[返回首页](../README.md) · [本轮修复](runtime-hardening.md) · [FRP 接入记录](frp-migration.md)

核对日期：**2026-09-19**。本页区分基础库已发布能力、本轮修复分支、自动验证与尚未进行的验收。

## 当前结论

第一阶段公共抽象已经可用，FRP 已实际复用 Runtime／tray／autostart／CI。Review 后的关键修复已提交推送，Kit 和 FRP 的三平台 CI 全绿。

当前发布目标为 **Kit v0.5.0**。v0.5.0 在 v0.4.x 配置与 Secure Config 基础上新增 Runtime Theme：4 个内置 fallback Theme、远程 manifest、SHA-256 校验和 last-known-good 完整快照。消费者启用 Runtime Theme 后，Theme 内容更新不再要求重新构建应用。

## 已完成的工作

| 范围 | 已完成内容 |
| --- | --- |
| 公共库基础 | 公开 Go Module、声明式 Config、Controller、TrayConfig、Hooks |
| 系统集成 | 三平台登录自启、Wails 单实例、窗口显示／隐藏、关闭策略 |
| UI 基础 | token、基础组件、导航样式、ui.Mount 与 Runtime Theme JS |
| 图标 | 跨平台 desktopkit icon：已有图片 normalize、Terminal / Monogram 确定性生成、PNG 原子写入与边界保护 |
| 配置路径 | 跨平台统一 XDG / `~/.config/<app-id>` 路径 API，不自动迁移旧消费者 |
| Secure Config | 系统凭据库托管随机主密钥；AES-256-GCM 加密 Secret/JSON，无明文降级 |
| Runtime Theme | 4 个内置 fallback；远程主题整包同步、SHA-256 校验、共享缓存、完整快照与离线恢复 |
| 构建工程 | Windows／Linux／macOS 并行 reusable workflow、wrapper、SHA256、可选 Release |
| 本轮 Runtime 修复 | macOS 主循环整合、托盘就绪／故障恢复、窗口并发保护、业务动作串行化 |
| 本轮接口与边界 | BeforeClose／SecondInstance／TrayError／AutoStartProvider，Linux quoting、UI FS、图标限制、Release 产物范围 |
| FRP 消费者 | 从自有桌面基础设施迁入 Kit；修复分支切换 HideSafe、固定公共依赖和 workflow SHA |
| 使用文档 | 中文快速接入、完整入口示例、Runtime、UI／图标、构建发布、升级说明与迁移记录 |

UI CSS 已存在不等于消费者页面已经统一；图标 CLI 已存在不等于产品脚本已经接入。

## 已通过的验证

| 验证 | 结果／证据 | 能说明什么 |
| --- | --- | --- |
| Kit 三平台 CI | [35371836841](https://github.com/wanstu/wails-desktop-kit/actions/runs/35371836841)，全部通过 | 三个平台的测试与 vet；新增 macOS 运行检查 |
| Windows 本地 | race／vet、production WebView2＋托盘启动退出通过 | 并发回归用例与一次真实原生生命周期 |
| Debian 13 容器 | WebKit 4.1 下 race 全通过；gio 启动参数测试通过 | Linux 编译／测试和实际 desktop Exec 解析 |
| macOS 运行检查 | Kit CI 的 production WebView＋托盘 smoke 通过 | 共用主循环下的一次启动和退出 |
| FRP 本地 | PowerShell wrapper 完整通过 | 前端检查、Go 测试、vet、Windows Wails 构建 |
| 使用文档示例 | 2026-09-19 独立 Module 的 Windows Wails 构建通过，9 页内部链接／代码块检查通过 | 快速接入入口与 Runtime／UI Go 示例可编译 |
| FRP 三平台 CI | [35371736550](https://github.com/wanstu/frp-client-manager/actions/runs/35371736550)，全部通过 | Windows amd64、Linux amd64、macOS universal 构建和产物上传 |

容器测试不等于 Linux 图形桌面验收。启动退出 smoke 不等于菜单所有动作都已验证。非 tag CI 中 Release 跳过，不等于真实发布路径已通过。

## 接下来优先完成

| 顺序 | 工作 | 完成标准 |
| --- | --- | --- |
| 1 | 真实桌面回归 | 三平台菜单顺序、显示／隐藏、关闭、重复启动；无可见宿主的 Linux 不丢窗口 |
| 2 | 自启动与 FRP 业务验收 | 登录启动、路径移动后重新启用；多 Profile，以及保留／停止 frpc 两种退出 |
| 3 | Review 与合并 | Kit 和 FRP 两个 PR 审查、处理结论并合并 |
| 4 | 版本与发布验收 | 选择新版本，验证消费者真实 tag Release、文件名／校验和／权限及下载运行，再完成正式版本切换 |

PR：
- [Kit：桌面生命周期与跨平台边界修复](https://github.com/wanstu/wails-desktop-kit/pull/1)
- [FRP：接入修复版](https://github.com/wanstu/frp-client-manager/pull/1)

## 后续功能／迁移

| 项目 | 当前状态 | 后续范围 |
| --- | --- | --- |
| FRP UI | 未迁移 | 逐页使用 Kit token／组件，检查控件状态与布局 |
| FRP icon script | 待迁移 | 改用 `desktopkit icon generate --symbol monogram --text F`，核对应用及托盘图标 |
| IME Lock | Runtime Theme 已接入 | 继续验证静默自启、重复自启不弹窗、平台专属逻辑边界 |
| AI Dev Manager | 未接入 | 验证后台服务退出选择、设置通知及复杂构建 wrapper |
| CodexPro+ | 未接入 | 验证第二次启动行为、额外 core build 和资源流程 |
| StatusNotifier host 探测 | 未实现 | 区分注册与可见宿主；宿主消失后的恢复与桌面兼容性 |
| 共享 build/package CLI | 未实现 | 根据更多消费者的共同需求确定 API |
| deb／安装包抽象 | 未实现 | 包结构、依赖、升级与卸载约定 |
| macOS signing/notarization | 未实现 | 签名、公证及正式分发验证 |

本轮按用户确认推进正式发布。尚未执行的人工桌面验收仍保留为待办；发布后再分批迁移其他产品，每个消费者分别验证 Runtime、产品退出语义和构建链路。
