# Wails Desktop Kit

面向 **Go + Wails v2** 桌面应用的公共基础库，集中维护窗口、托盘、登录自启、单实例、CSS 基础样式、图标处理和三平台 CI。业务项目继续拥有配置、进程管理和页面逻辑。

FRP Client Manager 已实际复用 Kit Runtime；IME Lock 已接入 Runtime Theme。AI Dev Manager、CodexPro+ 可按相同契约逐步迁移。

## 安装 v0.7.2

本文对应 **v0.7.2**。Linux 启用托盘时默认关闭窗口后保留后台进程，`desktopkit icon generate` 默认输出为 256×256。托盘点击行为交给平台托盘宿主与 systray 后端处理，不再在 Kit 中覆盖 Linux StatusNotifierItem 的点击语义。完整发布状态见 [Releases](https://github.com/wanstu/wails-desktop-kit/releases)。

~~~powershell
go get github.com/wanstu/wails-desktop-kit@v0.7.2
~~~

GitHub reusable workflow 同步固定：

~~~yaml
uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.7.2
~~~

Go Module 和 workflow 是两处独立版本引用，需要分别升级。可提交的依赖不要使用本地 replace。旧应用不会自动更新；发布的可执行文件也不会因为 Kit 更新而改变。

### 使用方需要改代码吗

采用命名字段 Config、声明式菜单，并直接传入 `*autostart.Manager` 的已有用法可以继续编译。新增 hook 都是可选项，通常无需修改业务逻辑。

旧版升级仍有需要核对的边界：AutoStart 字段类型由具体指针改为接口，依赖其具体类型的读取代码需适配；主动退出、隐藏时机及 UI 资源根目录需符合 [升级契约](docs/runtime-hardening.md)。早期 FRP 曾显式使用 HideSafe；v0.7.0 起 Kit 默认改为 HideAlways，但应用仍可显式选择任一策略。

后续升级以保持常规用法兼容为目标；需要使用方修改源码或改变行为的变更会在升级说明中明确列出。这里不承诺所有未来版本都可无条件替换。

| 版本 | 说明 |
| --- | --- |
| `v0.7.2` | 回滚 Linux 托盘点击覆盖与临时 systray fork；保留默认关闭到托盘和 256×256 图标生成 |
| `v0.7.1` | 尝试调整 Linux 托盘主激活行为，已在 v0.7.2 回滚 |
| `v0.7.0` | Linux 默认关闭到托盘；`icon generate` 默认 256×256；其中托盘点击尝试已在 v0.7.2 回滚 |
| `v0.6.1` | Packaging Pipeline 增加 Linux CI 真实包格式验收：dpkg-deb / tar / SHA256 全链路验证 |
| `v0.6.0` | Packaging Pipeline 支持 raw / deb / tar.gz、统一 SHA256、Debian 桌面元数据与通用 post-package hook；包含 v0.5.2 的 Debian 换行修复 |
| `v0.5.2` | 热修 v0.5.1 Debian control Description 最终换行，恢复 `.deb` consumer CI |
| `v0.5.1` | reusable workflow 新增可选 Linux `.deb` 产物与 SHA256 校验；默认关闭 |
| `v0.5.0` | Runtime Theme：4 个内置 fallback、远程 manifest、完整快照缓存、SHA-256 校验；Theme 更新无需重建消费者 |
| `v0.4.0` | 统一 `~/.config/<app-id>` 配置目录；新增 Secure Config，系统凭据库托管主密钥 + AES-256-GCM 密文 |
| `v0.3.1` | 新增确定性 Kit 家族图标生成器：Terminal / Monogram；保留原 normalize CLI |
| `v0.3.0` | 新增 Theme Pack 协议，与 light / dark / system 明暗模式正交；可选主题拆到独立模块 |
| `v0.2.2` | token 驱动的完整暗色主题契约；公共组件不再写死浅色颜色值 |
| `v0.2.1` | 修复 tag Release 产物文件名，产物自动包含版本号 |
| `v0.2.0` | 生命周期加固、跨平台边界修复和中文接入文档 |
| `v0.1.2` | 上一稳定基线，不包含本轮修复；[查看对应文档](https://github.com/wanstu/wails-desktop-kit/tree/v0.1.2) |
| `b314506`／对应伪版本 | 发布前 FRP 的验证版本；正式接入改用 v0.2.0 |

## 从这里开始

| 需求 | 文档 |
| --- | --- |
| 新建最小应用，或替换已有 Wails 入口 | [快速接入](docs/getting-started.md) |
| 配置窗口、托盘、生命周期、单实例、自启动 | [Runtime 使用指南](docs/runtime.md) |
| 挂载 CSS、选择组件类、统一图标 | [UI 与图标](docs/ui-and-icons.md) |
| 运行时下载、缓存、刷新 Theme Pack | [Runtime Theme](docs/runtime-theme.md) |
| 统一配置目录、保存密码／Token／私钥口令 | [配置目录与安全配置](docs/config-and-secrets.md) |
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
| `ui` | token、布局、表单、按钮、状态、对话框、导航 CSS 与 Theme Runtime JS | 页面结构、交互、数据和无障碍行为 |
| `theme` | 远程 manifest、Theme CSS 下载、SHA-256 校验、全局缓存与本地 AssetServer | 保存用户的 theme mode / pack 选择 |
| `icon`／`cmd/desktopkit` | 现有图片规范化；确定性生成圆角底色 + Terminal / Monogram 家族图标 | 产品选择 symbol / monogram、颜色与构建脚本中的调用位置 |
| `paths` | 跨平台统一 `$XDG_CONFIG_HOME/<app>`，默认 `~/.config/<app>` | 稳定 App ID |
| `secureconfig` | 系统凭据库托管主密钥；AES-256-GCM 加密 Secret / JSON | Secret 的逻辑 key 与业务生命周期 |
| `packaging` / `desktopkit package` | Linux raw / deb / tar.gz、桌面文件与图标安装、统一 SHA256；格式化 builder 结构便于后续 AppImage 等扩展 | 产品元数据、可选复杂格式 hook |
| reusable workflow | 三平台并行构建、Packaging Pipeline、通用 post-package hook、自动 SHA256、可选 Release | 应用构建目录、产品 wrapper、caller 权限 |

可以分阶段接入。使用 Runtime 不要求迁移 UI；使用 CSS 或 icon CLI 也不要求把业务入口改成 `desktopkit.Run`。

## 默认窗口行为

v0.7.0 起默认采用 `HideAlways`：

- Windows/macOS/Linux：启用托盘且托盘后端就绪后，关闭主窗口默认隐藏到托盘，不结束进程。
- Linux 托盘的单击、双击与菜单交互由桌面环境和 systray 后端决定，Kit 不覆盖宿主的点击语义。
- 自动启动：托盘就绪后按配置隐藏，因此启动阶段仍可能短暂看到窗口。
- 特殊退出：使用 `Controller.Quit()`；需要停止业务进程时，先完成业务清理再调用它。

如产品明确希望 Linux 点 × 直接退出，可显式设置 `HideSafe` 或 `HideNever`；`HideAlways` 仍要求托盘后端实际就绪。

## 当前验收状态

Kit 的 Runtime 基线已经通过 Windows／Linux／macOS CI 与 Windows/macOS 真实 WebView＋托盘启动退出检查。v0.5.0 在 v0.4.x 基础上增加 Runtime Theme 与完整离线快照；零值 ThemeConfig 仍保持关闭，因此已有应用升级 Kit 不会自动发起主题网络请求。

桌面菜单操作、真实登录自启、宿主丢失与消费者 tag Release 的验收仍待完成。详见 [进度与待办](docs/status.md)。
