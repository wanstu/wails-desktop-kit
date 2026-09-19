# FRP 迁移记录

[返回首页](../README.md) · [本轮修复](runtime-hardening.md) · [进度与待办](status.md)

状态核对：2026-09-19。FRP 已是实际消费者，本页记录完成情况与验收缺口，不再作为尚未开始的迁移目标。

## 两个阶段

| 阶段 | 依赖／提交 | 状态 |
| --- | --- | --- |
| 初次接入 | Kit v0.1.2；FRP 46ca18f、6f35e84 | 已进入原有主线并通过三平台 CI |
| 本轮加固接入 | Kit b314506 的公开伪版本；FRP b4bb2a6 | 修复分支已提交推送，PR 尚未合并 |

本轮 FRP 分支从 v1.2／504ef4a 建立，保留已有的按平台下载 frpc 功能。Go Module 使用 `v0.1.3-0.20260918165411-b3145064740c`，CI／Release 固定同一 Kit 完整 SHA，没有提交本地 replace。

[FRP 接入 PR](https://github.com/wanstu/frp-client-manager/pull/1) · [Kit 修复 PR](https://github.com/wanstu/wails-desktop-kit/pull/1)

## 已迁移

- Runtime：窗口配置、单实例、生命周期、启动参数。
- Tray：原生实现与生命周期交给 Kit，FRP 声明业务菜单。
- Autostart：删除业务仓库内重复的三平台实现，使用公共包。
- CI/CD：使用 reusable workflow，保留产品自己的 PowerShell／Unix build wrapper。
- 本轮升级增加 HideSafe，并修复 Windows wrapper 漏传 Wails 构建失败退出码的问题。

保留在 FRP 的业务：Profile、frpc 启停／恢复／下载、FRP 配置编辑、持久化以及两种退出动作。

## 托盘顺序与退出语义

菜单依次为：显示／隐藏、启动全部／停止全部／重启全部、登录自启、退出（保留 frpc）、退出并停止 frpc。Kit 在分组之间添加分隔线，FRP 设置 DisableQuit 以移除重复的默认退出项。

| 操作 | 修复分支行为 |
| --- | --- |
| Windows/macOS 关闭窗口，托盘正常 | 隐藏到托盘 |
| Linux 关闭窗口，或托盘不可用时关闭窗口 | 退出管理器；FRP 当前 Shutdown 不停止 frpc |
| 退出（保留 frpc） | 直接通过 Controller.Quit 退出管理器 |
| 退出并停止 frpc | 先停止全部；成功后退出，失败则报告错误并保留管理器 |
| 托盘初始化／运行失败 | Kit 恢复主窗口 |
| Linux 自启动 | 保持窗口可见 |

从 HideAlways 改为 HideSafe 是明确的行为调整：Linux 不再默认隐藏。它用于防止没有托盘宿主时窗口不可找回，不能把这一改变描述为所有平台行为完全未变。

## 已验证

- 本地 Windows build wrapper：前端语法／DOM 检查、Go 测试、vet、Wails production build 通过。
- [消费者 CI 35371736550](https://github.com/wanstu/frp-client-manager/actions/runs/35371736550)：Windows amd64、Linux amd64、macOS universal 构建及产物上传全部成功。
- 普通 CI 的 Publish GitHub Release 按预期跳过。

Kit 的 race 测试与原生启动退出检查见 [总进度](status.md)。这些结果不等于 FRP 每个业务操作都经过人工验收。

## 仍需完成

- 在真实桌面检查菜单、隐藏／恢复、重复启动与登录自启。
- 用实际或隔离测试进程验证“保留”和“停止后退出”，确认多 Profile 可独立运行和恢复。
- 用本轮新工作流完成一次真实消费者 tag Release。
- 完成 review、合并及版本切换。

FRP UI 尚未改用 Kit CSS；原有 icon build script 尚未正式切换为 desktopkit icon。这两项应单独迁移、单独验证。
