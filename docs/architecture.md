# 架构与抽象边界

[返回首页](../README.md) · [Runtime](runtime.md) · [进度与待办](status.md)

Kit 的目标是让多个 Wails 应用复用桌面基础设施。FRP 已完成第一轮实际接入，证明这条抽象路径可用；它是否充分覆盖其他产品，仍需后续消费者验证。

## 职责划分

| Kit 负责 | 业务项目负责 |
| --- | --- |
| Wails 生命周期组合、窗口控制、单实例 | 业务对象、配置存储和进程／服务管理 |
| 原生托盘生命周期、菜单映射、失败恢复 | 菜单动作、checkbox 状态、退出前的业务清理 |
| 三平台登录自启存储 | 是否启用、参数、应用自己的状态事件 |
| CSS token、基础组件与导航样式；4 个内置 fallback Theme；远程 Theme 下载/缓存 | 页面信息结构、交互、路由、业务数据与用户主题选择持久化 |
| 图标规范化 | 图标源文件及调用 CLI 的构建步骤 |
| 固定三平台桌面矩阵、产物和可选发布 | 产品额外检查、CLI／额外架构、专属安装和签名流程 |

FRP 的连接配置、frpc 生命周期，ADM 的 Gateway/MCP，IME 修复逻辑和 Codex 工作区进程都留在各自仓库。Kit 不接管业务状态。

## 模块组合

- 根包 `desktopkit`：通过 Config／Hooks／Controller 组合 Wails 和系统集成。
- `autostart`：独立的 Supported／Enabled／SetEnabled API，也可在未使用 Runtime 的应用中调用。
- `ui`：嵌入 CSS 与 FS overlay，不依赖前端框架。
- `icon` 与 `cmd/desktopkit`：图片规范化库及 CLI，目前没有共享 build/package 命令。
- reusable workflow：面向一个标准 Wails 桌面应用，允许产品 wrapper 补充步骤。

API 采用显式配置和小接口。当前没有插件注册系统或通用任务图；复杂产品可以只采用所需模块。

## 生命周期与线程

Wails 拥有应用主循环。Windows/Linux 托盘后端在专用 OS 线程上运行；macOS 创建和移除 NSStatusItem 时派发到现有 AppKit 主线程，不再运行另一套 NSApplication 循环。

原生菜单回调将业务动作交给串行队列。Kit 状态锁不会跨越 Wails 原生窗口调用；完成顺序颠倒时，会重新应用最新的显示／隐藏意图。这样既避免持锁等待主线程，也避免较早的隐藏覆盖后来的显示。

退出会停止接受新托盘工作并请求原生清理。业务回调可能仍在执行，Kit 无法强制终止任意 Go 回调；业务应有自己的超时和取消机制。详细 hook 顺序见 [Runtime](runtime.md)。

## 可恢复性优先

配置允许隐藏、托盘后端可用、桌面上存在可见入口是不同条件。Windows 检查 shell 图标位置；macOS 检查原生状态项创建；Linux 检查本进程的 StatusNotifierItem 注册。

Linux 的 `HideSafe` 仍保留保守语义，但 v0.7.0 的默认窗口策略改为 `HideAlways`。托盘点击行为继续由桌面环境与 systray 后端决定；托盘后端仍需完成就绪探测后才允许隐藏窗口。

## 扩展点的用途

| 扩展点 | 解决的问题 |
| --- | --- |
| Items／FooterItems／DisableQuit | 普通业务菜单与多种退出模式 |
| AutoStartProvider | 复用产品自己的设置写入及状态通知 |
| SecondInstance | 保留第二次启动参数，处理静默自启、文件或 URL |
| BeforeClose | 未保存状态确认，允许业务取消关闭或退出 |
| TrayError | 业务记录或展示托盘故障，窗口恢复由 Kit 先执行 |
| build-command-* | 保留产品资源准备和校验，无需复制三平台矩阵 |

后续消费者如果需要尚未暴露的 Wails 能力，应先确认是否属于多个应用共同需求，再扩展 API。不要为了保留某个产品的全部历史入口，直接把所有业务细节搬进 Config。
