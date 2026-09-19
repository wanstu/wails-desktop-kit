# Runtime 使用指南

[返回首页](../README.md) · [快速接入](getting-started.md)

本文描述首页指定的修复版本。`SecondInstance`、`Hooks.BeforeClose`、`Hooks.TrayError`、`AutoStartProvider` 及托盘就绪保护不属于 v0.1.2 的原有契约。

## 应用配置

| 字段 | 用法 |
| --- | --- |
| `ID` | 必填，稳定且应用间唯一；启用单实例时用作锁 ID |
| `Title` | 必填，用于窗口标题和默认托盘提示 |
| `Assets` | 必填，应用的 `fs.FS`；挂载 Kit UI 时必须先对齐到含 index.html 的根目录 |
| `Bind` | 原有 Wails 业务对象，例如 `[]interface{}{app}` |
| `Launch` | 启动参数，主要是 `AutoStart` |
| `Window` | 尺寸、背景和隐藏策略 |
| `Tray` | 图标、默认菜单、业务菜单、自启动 provider |
| `Hooks` | 生命周期及托盘错误处理 |
| `SingleInstance` | 默认 false，需要显式开启 |
| `SecondInstance` | 第二次启动回调；只在启用单实例时生效 |

优先从 `DefaultWindowConfig()` 修改配置：默认尺寸 1024×720，最小 720×520，`HideSafe`，自启动时允许延后隐藏。

直接传 `WindowConfig{}` 会补齐尺寸和隐藏策略，但布尔字段 `StartHiddenOnAutoStart` 仍为 false；它与 `DefaultWindowConfig()` 不完全等价。

`ParseLaunchOptions(os.Args[1:])` 只处理标准自启动参数，未知选项／额外位置参数会报错。带 CLI 子命令或文件参数的产品应自行解析，再构造 `LaunchOptions{AutoStart: ...}`。

## 隐藏、恢复和退出

下表中的“关闭”均指业务 `BeforeClose` 未取消操作的情况。

| 策略 | Windows/macOS | Linux |
| --- | --- | --- |
| `HideSafe` | 托盘就绪后关闭到托盘；不可用时允许退出 | 不隐藏，关闭时允许退出 |
| `HideAlways` | 仍需托盘后端就绪 | 后端就绪后允许隐藏；可见宿主由应用负责确认 |
| `HideNever` | 不隐藏，关闭时允许退出 | 不隐藏，关闭时允许退出 |

这些规则也约束 `Controller.HideWindow()`，并非仅约束关闭按钮。没有启用托盘时不会关闭到托盘。`CanHideToTray` 只判断配置与平台策略，不是运行时托盘可见性探测接口。

自动隐藏需要同时满足：`Launch.AutoStart`、`StartHiddenOnAutoStart`、允许隐藏的策略和托盘就绪。窗口初始可见，可能短暂显示；用户主动显示后，会取消尚未执行的自动隐藏。

| Controller 方法 | 行为 |
| --- | --- |
| `Context()` | 返回 Wails context；尚未启动／停止后可为 nil |
| `ShowWindow()` | 显示并取消最小化，取消待执行的自动隐藏 |
| `HideWindow()` | 在当前策略允许且托盘就绪时隐藏 |
| `Quit()` | 标记主动退出，使这次操作绕过关闭到托盘；仍可被 BeforeClose 取消 |
| `ShowError(title, err)` | 恢复窗口并显示错误对话框；nil error 不处理 |

需要退出时优先使用 Kit 回调提供的 `Controller.Quit()`。直接调用 Wails 的 Quit 不会设置 Kit 的主动退出标记，在允许隐藏的情况下可能被解释为关闭到托盘。

## 托盘菜单

配置顺序为：标准显示／隐藏、`Items`、`AutoStart`、`FooterItems`、标准退出。Kit 按配置添加组间分隔线；组内可使用 `Separator()`。

- `Enabled: true` 时必须提供有效 PNG `Icon`。
- `DisableShowHide` 移除默认显示／隐藏项。
- `DisableQuit` 移除默认退出项，适合产品自行提供退出模式。
- `ShowLabel`、`HideLabel`、`LaunchAtLoginLabel`、`QuitLabel` 可覆盖默认中文文字。
- `Action` 返回错误后会显示错误；`TrayItem.ErrorTitle` 可提供业务错误前缀。
- `Checkbox` 需同时提供 Get 与 Set；Get 读取当前状态，Set 接收切换后的值。

下面是可放入构造 Config 的函数。它保留两种退出语义，停止失败时不退出：

~~~go
func exitTray(icon []byte, stopAll func() error) desktopkit.TrayConfig {
	return desktopkit.TrayConfig{
		Enabled:     true,
		Icon:        icon,
		DisableQuit: true,
		FooterItems: []desktopkit.TrayItem{
			desktopkit.Action("退出（保留后台进程）", func(c *desktopkit.Controller) error {
				c.Quit()
				return nil
			}),
			desktopkit.Action("停止后台进程并退出", func(c *desktopkit.Controller) error {
				if err := stopAll(); err != nil {
					return err
				}
				c.Quit()
				return nil
			}),
		},
	}
}
~~~

业务动作和 checkbox 读取在同一 worker 上串行运行；正在等待或执行的同一动作被重复点击时，重复请求会丢弃。状态会定期刷新，动作完成后也会刷新。显示／隐藏不等待业务动作队列。

回调必须能结束，耗时任务应有自己的取消或超时控制。Shutdown 不等待正在执行的业务回调结束，也无法强行终止它。前端与托盘同时访问的业务状态仍需应用自行同步。

## 生命周期 hook

| Hook | 顺序／语义 |
| --- | --- |
| `Startup(ctx)` | Kit 先保存 context，再执行业务启动 |
| `DomReady(ctx)` | 先执行业务回调，再启动托盘；此时不能假定托盘已就绪 |
| `BeforeClose(ctx) bool` | 先于隐藏／退出决策；true 取消本次操作，false 继续；主动退出同样会调用它 |
| `Shutdown(ctx)` | Kit 先停止接收托盘任务、请求原生清理并停用 Controller，再执行业务关闭 |
| `TrayError(ctx, err)` | 托盘初始化／运行失败时先恢复窗口再通知；未配置时使用默认错误弹窗 |

应用根据自身需求在 Shutdown 中清理业务资源。Kit 不会自行停止 frpc、Gateway 或其他业务子进程。macOS 原生托盘清理通过主线程派发，Shutdown 回调不是所有原生清理均已完成的等待屏障。

## 第二次启动

默认行为是显示并取消最小化。下面示例只适用于首个进程也使用标准自启动参数的应用，用于忽略重复登录启动，同时允许手动启动唤醒窗口：

~~~go
func onSecondInstance(c *desktopkit.Controller, data options.SecondInstanceData) {
	launch, err := desktopkit.ParseLaunchOptions(data.Args)
	if err == nil && launch.AutoStart {
		return
	}
	c.ShowWindow()
}
~~~

需导入 `github.com/wailsapp/wails/v2/pkg/options`，并在 Config 设置 `SecondInstance: onSecondInstance`。回调同时提供 `WorkingDirectory`。Wails v2.15.0 实际将它设为可执行文件所在目录，不应直接当作启动者当前工作目录。处理文件／URL 的产品应使用自己的参数解析器，而不是上面的简化策略。

## 登录自启

`autostart.New(Config)` 只创建管理器；`SetEnabled(true)` 才写入系统设置。

| 平台 | 保存位置 | 生效范围 |
| --- | --- | --- |
| Windows | HKCU 的 `Software\Microsoft\Windows\CurrentVersion\Run`，值名为 ID | 当前用户登录 |
| Linux | `$XDG_CONFIG_HOME/autostart/<id>.desktop`；未设置时为 `~/.config/autostart/` | 当前用户图形会话 |
| macOS | `~/Library/LaunchAgents/<id>.plist` | 下一次登录加载 |

`ExecutablePath` 默认是当前可执行文件；`Arguments` 是参数数组，不要提前拼接成 shell 命令。Linux 会处理 desktop entry 和 Exec 参数的两层转义，拒绝换行／NUL 参数及不支持的可执行路径；非空 `XDG_CONFIG_HOME` 必须为绝对路径。

`Enabled()` 检查持久化内容是否符合期望路径与参数，不保证桌面环境一定执行了该项。移动可执行文件后，应从新位置重新启用；Kit 不会在普通启动时悄悄修复旧路径。

macOS 当前只管理 plist，不调用 `launchctl bootstrap/bootout`。勾选不会立即再启动一个进程，取消勾选也不会停止当前应用。

### 使用业务自己的设置与事件

`TrayConfig.AutoStart` 接受以下接口；原有 `*autostart.Manager` 可以直接使用：

~~~go
type AutoStartProvider interface {
	Supported() bool
	Enabled() (bool, error)
	SetEnabled(bool) error
}
~~~

需要即时刷新前端时，可包装 Manager：SetEnabled 成功后发送应用自己的状态事件；前端设置页也走同一业务入口。Kit 不定义产品事件名，不替产品维护设置副本。

AutoStart 不为 nil 时必须同时启用托盘。仅需独立自启动功能的 CLI／设置页可以直接使用 autostart 包，无需启用整个 Runtime。
