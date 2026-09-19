# 本轮修复与升级说明

[返回首页](../README.md) · [Runtime 用法](runtime.md) · [进度与待办](status.md)

这些变更位于 `fix/runtime-hardening`，不包含在已发布的 v0.1.2 中。完整示例使用首页固定的修复提交；尚未创建新的稳定版本。

## 修复内容

| 范围 | 原问题／边界 | 当前处理 |
| --- | --- | --- |
| macOS 托盘 | 独立原生事件循环与 Wails 主循环的所有权不清晰 | 在现有 AppKit 主线程创建／更新／移除状态项，不再单独运行或停止 NSApplication |
| 隐藏与恢复 | 初始化失败、晚到的隐藏请求可能使窗口不可恢复 | 等待托盘就绪、失败恢复、主动显示取消待隐藏；并发完成顺序颠倒时重新应用最新意图 |
| 托盘回调 | 业务操作占用原生回调路径 | 有界串行 worker 执行业务动作及状态读取，重复待执行动作丢弃 |
| 生命周期 | 业务缺少关闭拦截、第二次启动参数和错误扩展点 | 增加 BeforeClose、SecondInstance、TrayError，明确 hook 顺序 |
| 自启动扩展 | TrayConfig 绑定具体 Manager | 改为 AutoStartProvider 接口，现有 Manager 仍可直接传入 |
| Linux 自启动 | Exec 参数和 desktop 字符串的两层转义容易混淆 | 分层转义，以 gio 实际启动核验特殊字符；拒绝不支持的输入 |
| UI FS | 根目录／子目录与 Wails 资源路由边界不清晰 | 补齐目录读取，验证 fs 路径及实际 Wails 资源处理器 |
| 图标工具 | 解码／分配缺少尺寸和非有限值限制，写失败可能破坏已有输出 | 先检查头部及边界，临时输出成功后替换；保留 premultiplied 插值并增加透明边缘测试 |
| Release | 下载本次 run 的全部 artifact，混入无关产物 | 固定三个平台 artifact，并检查核心文件与 SHA256；build job 明确只读 |

Linux 的 HideSafe 仍保持保守策略，没有完成可见 StatusNotifier host 探测。macOS LaunchAgent 仍采用“写入配置、下次登录生效”，没有新增 launchctl 即时加载／卸载。

## 升级时需要检查的行为变化

1. 自动启动不再以“窗口一开始就隐藏”为前提：托盘就绪后才隐藏，可能短暂显示。
2. HideAlways 仍要求托盘后端就绪；HideNever 及 Linux HideSafe 下的手动 HideWindow 不生效。
3. 主动退出菜单使用 Controller.Quit；业务 BeforeClose 返回 true 仍可取消退出。
4. 多次点击同一忙碌菜单项不会积累重复执行；不要依赖重复点击排队实现业务计数。
5. ui.Mount 前先把 app FS 对齐到 index.html 所在目录，desktopkit/ 是保留目录。
6. 超大图标、NaN／Inf fill、Linux 不支持的 Exec 输入会返回错误；构建脚本应传播失败。

托盘 getter、setter 与动作使用一个 worker，但前端可能同时访问业务状态，应用自己的锁仍有必要。Shutdown 停止接收新任务，不是等待任意业务 callback 结束的 join。

## 升级步骤

1. 固定 Kit Go Module 与 workflow 到同一已审查代码版本。
2. 检查产品的隐藏策略、第二次启动参数、退出动作及生命周期 hook。
3. 验证资产根目录、图标源和 wrapper 的退出码。
4. 先跑 Kit 与消费者测试／构建，再在真实桌面核对操作。
5. 完成合并、版本选择与真实 tag Release 验收后，再对外宣称新的稳定基线。

FRP 已完成前 3 项及自动构建验证，采用 HideSafe，并保留两种业务退出模式。UI／图标脚本不在此次 FRP 升级范围内。

## 提交与验证证据

| 提交 | 内容 |
| --- | --- |
| `3056f58` | 托盘生命周期、窗口恢复与应用扩展点 |
| `b314506` | 跨平台边界、并发回归测试和桌面运行验收 |
| `eb20724` | macOS smoke 补齐 Wails 系统框架链接参数 |
| FRP `b4bb2a6` | 升级修复依赖、HideSafe、Windows 构建退出码 |

- [Kit CI 35371836841](https://github.com/wanstu/wails-desktop-kit/actions/runs/35371836841)：三平台测试／vet；macOS production WebView＋托盘启动退出。
- 本地 Windows race／vet、WebView2＋托盘启动退出通过。
- Debian 13 的 `go test -tags webkit2_41 -race ./...` 通过，包含 gio 实际解析 Exec 参数。
- [FRP CI 35371736550](https://github.com/wanstu/frp-client-manager/actions/runs/35371736550)：三平台 production 构建及 artifact 上传通过。

这些检查未替代真实菜单点击、登录自启、宿主消失及消费者 tag 发布。剩余验收集中维护在 [进度与待办](status.md)。
