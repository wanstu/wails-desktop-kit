# Wails Desktop Kit

面向 **Go + Wails v2** 桌面应用的公共基础库，集中维护窗口、托盘、登录自启、单实例、CSS 基础样式、图标处理和三平台 CI。业务项目继续拥有配置、进程管理和页面逻辑。

首个实际消费者是 [FRP Client Manager](https://github.com/wanstu/frp-client-manager)。IME Lock、AI Dev Manager、CodexPro+ 尚未接入。

## 安装 v0.2.0

本文对应 **v0.2.0**，包含本轮 Runtime 与跨平台边界修复。完整发布状态见 [Releases](https://github.com/wanstu/wails-desktop-kit/releases)。

~~~powershell
go get github.com/wanstu/wails-desktop-kit@v0.2.0
~~~

GitHub reusable workflow 同步固定：

~~~yaml
uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.2.0
~~~

Go Module 和 workflow 是两处独立版本引用，需要分别升级。可提交的依赖不要使用本地 replace。旧应用不会自动更新；发布的可执行文件也不会因为 Kit 更新而改变。

### 使用方需要改代码吗

采用命名字段 Config、声明式菜单，并直接传入 `*autostart.Manager` 的已有用法可以继续编译。新增 hook 都是可选项，通常无需修改业务逻辑。

本轮仍有需要核对的边界：AutoStart 字段类型由具体指针改为接口，依赖其具体类型的读取代码需适配；主动退出、隐藏时机及 UI 资源根目录需符合 [升级契约](docs/runtime-hardening.md)。FRP 将 HideAlways 改为 HideSafe 是主动调整 Linux 产品行为，不是适配新 API 的必要改动。

后续升级以保持常规用法兼容为目标；需要使用方修改源码或改变行为的变更会在升级说明中明确列出。这里不承诺所有未来版本都可无条件替换。

| 版本 | 说明 |
| --- | --- |
| `v0.2.0` | 本轮发布版本，包含生命周期加固、边界修复和中文接入文档 |
| `v0.1.2` | 上一稳定基线，不包含本轮修复；[查看对应文档](https://github.com/wanstu/wails-desktop-kit/tree/v0.1.2) |
| `b314506`／对应伪版本 | 发布前 FRP 的验证版本；正式接入改用 v0.2.0 |

## 从这里开始

| 需求 | 文档 |
| --- | --- |
| 新建最小应用，或替换已有 Wails 入口 | [快速接入](docs/getting-started.md) |
| 配置窗口、托盘、生命周期、单实例、自启动 | [Runtime 使用指南](docs/runtime.md) |
| 挂载 CSS、选择组件类、统一图标 | [UI 与图标](docs/ui-and-icons.md) |
| 本地构建、三平台 CI、权限和 Release | [构建与发布](docs/build-release.md) |
| 判断哪些代码应放入 Kit | [架构边界](docs/architecture.md) |
| 对照首个消费者的实际接入 | [FRP 迁移记录](docs/frp-migration.md) |
| 升级时检查行为变化 | [本轮修复说明](docs/runtime-hardening.md) |
| 了解源码兼容与本地目录范围 | [兼容与本地开发](docs/compatibility-and-local-development.md) |
| 已完成、验证证据和剩余工作 | [进度与待办](docs/status.md) |

## 提供哪些能力

| 模块 | 可复用内容 | 业务仍需提供 |
| --- | --- | --- |
| 根包 `desktopkit` | Wails 生命周期、单实例、窗口控制、托盘菜单与失败恢复 | 应用 ID、标题、资源、绑定对象、业务回调 |
| `autostart` | Windows Run、Linux desktop entry、macOS LaunchAgent | 稳定 ID、启动参数、启用／禁用入口 |
| `ui` | token、布局、表单、按钮、状态、对话框和导航 CSS | 页面结构、交互、数据和无障碍行为 |
| `icon`／`cmd/desktopkit` | 透明边缘裁剪、等比缩放、方形 PNG 输出 | 原始图标、构建脚本中的调用位置 |
| reusable workflow | 三平台并行构建、产物归档、SHA256、可选 Release | 应用构建目录、产品 wrapper、caller 权限 |

可以分阶段接入。使用 Runtime 不要求迁移 UI；使用 CSS 或 icon CLI 也不要求把业务入口改成 `desktopkit.Run`。

## 默认窗口行为

修复版默认采用 `HideSafe`：

- Windows/macOS：托盘就绪后才允许关闭到托盘；故障时恢复窗口。
- Linux：暂不自动隐藏，也不响应隐藏请求。关闭窗口会正常退出应用，业务退出清理由应用决定。
- 自动启动：Windows/macOS 先显示，托盘就绪后按配置隐藏，因此可能短暂看到窗口。
- 特殊退出：使用 `Controller.Quit()`；需要停止业务进程时，先完成业务清理再调用它。

Linux 的 StatusNotifierItem 注册成功不等于桌面有可见托盘宿主。`HideAlways` 需要应用承担这一环境前提；它仍不会绕过托盘后端就绪检查。

## 当前验收状态

Kit 与 FRP 修复分支的 Windows／Linux／macOS CI 已通过，Windows 和 macOS 的真实 WebView＋托盘启动退出检查已通过。修复及文档纳入 v0.2.0 发布，FRP 正在切换正式版本依赖。

桌面菜单操作、真实登录自启、宿主丢失与消费者 tag Release 的验收仍待完成。详见 [进度与待办](docs/status.md)。
