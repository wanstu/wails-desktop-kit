# 进度、证据与待办

[返回首页](../README.md) · [本轮修复](runtime-hardening.md) · [FRP 接入记录](frp-migration.md)

核对日期：**2026-09-28**。本页区分已发布能力、自动验证与下一阶段工作。

## 当前结论

Kit 的 Runtime、Theme、配置、安全存储、Build Metadata 与三平台 Packaging 基线已经可用。**Kit v0.10.0** 已完成发布收口，本轮把 Windows 安装从“只有 portable exe”提升为 Kit 一等能力。

Windows Installer v1 已完成：`desktopkit package windows` 直接包装业务已经构建好的 exe，不二次执行 `wails build`；reusable workflow 可同时产出 Portable EXE、NSIS Setup、Portable ZIP 与 SHA256。默认 user scope，支持 machine scope、开始菜单／桌面快捷方式、Windows 卸载注册、`/S` 静默安装与卸载，并明确不递归删除业务 AppData／数据库等持久数据。Know Me 已完成真实 Release、安装、运行、覆盖安装与卸载验收，下一阶段进入 Updater。

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
| 构建工程 | Windows／Linux／macOS 并行 reusable workflow、wrapper、Linux raw/deb/tar.gz、Windows NSIS Setup、Portable/Setup/ZIP、通用 post-package hook、SHA256、可选 Release |
| 本轮 Runtime 修复 | macOS 主循环整合、托盘就绪／故障恢复、窗口并发保护、业务动作串行化 |
| 本轮接口与边界 | BeforeClose／SecondInstance／TrayError／AutoStartProvider，Linux quoting、UI FS、图标限制、Release 产物范围 |
| FRP 消费者 | 从自有桌面基础设施迁入 Kit；修复分支切换 HideSafe、固定公共依赖和 workflow SHA |
| 使用文档 | 中文快速接入、完整入口示例、Runtime、UI／图标、构建发布、升级说明与迁移记录 |

UI CSS 已存在不等于消费者页面已经统一；图标 CLI 已存在不等于产品脚本已经接入。

## 已通过的验证

| 验证 | 结果／证据 | 能说明什么 |
| --- | --- | --- |
| Kit v0.10.0 Installer E2E CI | [36323530206](https://github.com/wanstu/wails-desktop-kit/actions/runs/36323530206)，全部通过 | Windows/Linux/macOS 测试与 vet；Windows 真实 NSIS 静默安装/卸载；reusable workflow 实际生成并重新下载校验 Portable EXE / Setup EXE / ZIP / SHA256 |
| Know Me 真实消费者 | [36326162758](https://github.com/wanstu/know_me/actions/runs/36326162758)，`v0.1.6-rc.4` Release 全绿 | 真实产品 wrapper 生成 Setup；本机 user-scope 安装成功；安装 exe 与 Portable SHA256 一致；运行数据落 `~/.config/know-me`；覆盖安装保留数据；静默卸载删除程序/快捷方式/注册表但保留用户数据 |
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
| 1 | 正式 v0.10.0 | 默认 helper / 文档固定正式 tag，发布稳定版本 |
| 2 | Know Me 切正式 Kit | 从 `v0.10.0-rc.1` workflow 切到 `v0.10.0`，保留 Windows Setup 为必需 Release 资产 |
| 3 | Updater Core | 实现版本比较、Release provider、下载、SHA256 Verify；不直接操作 GUI |
| 4 | Windows 自动更新 | 已安装实例调用 Setup 静默升级并重启；Portable 只检查/下载，不直接自覆盖 |

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
| 共享 build/package CLI | Windows/Linux 已实现 | `desktopkit package linux` 支持 raw / deb / tar.gz；`desktopkit package windows` 支持基于现成 exe 的 NSIS Setup |
| Windows Installer | 已完成 | user/machine scope、静默安装/卸载、卸载注册、快捷方式、覆盖升级与 E2E CI；Know Me 真实消费者验收通过 |
| Updater | 下一阶段 | GitHub Release provider 起步，Check / Download / SHA256 Verify / Windows Install / Restart；保持 provider 可替换 |
| AppImage / DMG / MSI | 扩展点已就绪 | 当前用 post-package hook；成熟后升级为 Kit 内置 builder |
| macOS signing/notarization | 未实现 | 签名、公证及正式分发验证 |

本轮按用户确认推进正式发布。尚未执行的人工桌面验收仍保留为待办；发布后再分批迁移其他产品，每个消费者分别验证 Runtime、产品退出语义和构建链路。
