# 快速接入

[返回首页](../README.md) · [Runtime](runtime.md) · [构建与发布](build-release.md)

本文使用首页指定的修复版本，目标是先跑通一个含托盘、登录自启和 Kit CSS 的最小应用。已有应用可直接阅读最后的迁移步骤。

示例已于 2026-09-19 从本文提取到独立 Go Module，使用公开修复版依赖完成 Windows Wails production build；Runtime 与 UI 页面中的 Go 片段也已编译检查。示例运行仍需准备自己的 PNG 图标和对应平台环境。

## 1. 准备环境

当前 Kit 的 `go.mod` 要求 Go 1.26；Wails 版本为 v2.15.0。

| 平台 | 构建／运行前提 |
| --- | --- |
| Windows | Go、Wails CLI、WebView2 Runtime；执行 race 测试还需可用的 C 编译器 |
| Linux | C 编译器、GTK3、WebKitGTK 4.1；在图形会话中运行应用 |
| macOS | Xcode Command Line Tools、CGO；使用 Wails 构建器处理系统框架链接 |
| 有前端构建步骤的项目 | 再按项目要求安装 Node.js；下述静态 HTML 示例没有前端打包步骤 |

在 PowerShell 中安装 CLI，然后新建空项目：

~~~powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
New-Item -ItemType Directory -Path desktop-demo
Set-Location desktop-demo
go mod init example.com/desktop-demo
go get github.com/wanstu/wails-desktop-kit@v0.2.2
New-Item -ItemType Directory -Path assets, frontend, build
~~~

如果找不到 `wails`，确认 Go 的 bin 目录已加入 PATH；可用 `go env GOPATH` 查看其父目录。

## 2. 准备资源与构建配置

将自己的 **有效 PNG** 图标保存为 `assets/appicon.png`。托盘图标必须是 PNG，修复版限制每边最多 4096 像素。可用 [icon CLI](ui-and-icons.md) 先规范化。

~~~powershell
Copy-Item assets/appicon.png build/appicon.png
~~~

这里两处路径职责不同：`assets/appicon.png` 供 Go 嵌入托盘；`build/appicon.png` 供 Wails 打包应用图标。

在项目根目录创建 `wails.json`：

~~~json
{
  "name": "desktop-demo",
  "outputfilename": "desktop-demo",
  "frontend:dir": "frontend",
  "wailsjsdir": "frontend"
}
~~~

创建 `frontend/index.html`：

~~~html
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Desktop Demo</title>
  <link rel="stylesheet" href="/desktopkit/tokens.css">
  <link rel="stylesheet" href="/desktopkit/base.css">
  <link rel="stylesheet" href="/desktopkit/components.css">
  <link rel="stylesheet" href="/desktopkit/navigation.css">
</head>
<body class="dk-app-shell">
  <main class="dk-content">
    <section class="dk-panel">
      <div class="dk-panel-heading">
        <h1>桌面应用已启动</h1>
        <span class="dk-status-pill is-success">就绪</span>
      </div>
      <p class="dk-muted">使用托盘菜单显示窗口、切换登录自启或退出。</p>
    </section>
  </main>
</body>
</html>
~~~

## 3. 创建完整入口

将下面的代码保存为 `main.go`。示例不依赖未定义的业务对象；它不默认开启系统自启动，用户勾选托盘菜单后才写入设置。

~~~go
package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	desktopkit "github.com/wanstu/wails-desktop-kit"
	"github.com/wanstu/wails-desktop-kit/autostart"
	kitui "github.com/wanstu/wails-desktop-kit/ui"
)

//go:embed all:frontend
var frontend embed.FS

//go:embed assets/appicon.png
var appIcon []byte

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	launch, err := desktopkit.ParseLaunchOptions(os.Args[1:])
	if err != nil {
		return err
	}
	assets, err := fs.Sub(frontend, "frontend")
	if err != nil {
		return err
	}
	login, err := autostart.New(autostart.Config{
		ID:          "com.example.desktop-demo",
		DisplayName: "Desktop Demo",
		Arguments:   []string{"--autostart"},
	})
	if err != nil {
		return err
	}
	window := desktopkit.DefaultWindowConfig()
	window.Width = 1080
	window.Height = 720

	return desktopkit.Run(desktopkit.Config{
		ID:             "com.example.desktop-demo",
		Title:          "Desktop Demo",
		Assets:         kitui.Mount(assets),
		Launch:         launch,
		Window:         window,
		SingleInstance: true,
		Tray: desktopkit.TrayConfig{
			Enabled:   true,
			Icon:      appIcon,
			AutoStart: login,
		},
	})
}
~~~

自己的应用应换成独有、稳定的 ID。同一个应用后续升级保持该 ID，以维持单实例锁和自启动项的身份。

## 4. 构建并检查

Windows，在项目根目录执行：

~~~powershell
go mod tidy
wails build -clean
./build/bin/desktop-demo.exe
~~~

Linux：

~~~bash
wails build -clean -tags webkit2_41
./build/bin/desktop-demo
~~~

macOS：

~~~bash
wails build -clean -platform darwin/universal
open ./build/bin/desktop-demo.app
~~~

先确认页面样式和托盘菜单正常，再检查重复启动能唤醒窗口。登录自启应在应用最终安装位置测试，避免记录临时构建目录。

`wails build` 的运行目录必须包含 `wails.json`。不要把普通 `go build` 当成完整桌面打包步骤。

## 5. 接入已有应用

按下面顺序迁移，每一步都可以独立验收：

1. 保留现有 Wails 配置、业务对象和前端构建脚本，固定 Kit 依赖版本。
2. 用 `Config.Bind` 传入原有业务对象，把原有 Startup／DomReady／Shutdown 交给 `Hooks`。
3. 把托盘业务操作转换为 `Action`／`Checkbox`，特殊退出动作放入 `FooterItems` 并设置 `DisableQuit`。
4. 明确单实例、自启动参数和关闭行为。默认使用 `HideSafe`；需要自定义第二次启动行为时使用 `SecondInstance`。
5. 删除已被 Kit 接管的原生托盘循环、重复自启动实现和旧单实例 IPC，避免两套逻辑同时运行。
6. 用 reusable workflow 替换平台矩阵；保留产品 wrapper 中的业务检查。
7. 最后再迁移 CSS 和图标脚本，分别检查页面和打包结果。

如果使用 Vite 等工具，嵌入实际构建目录，例如 `frontend/dist`，并对该目录做 `fs.Sub` 后再 `kitui.Mount`。保留原有 `frontend:install`／`frontend:build` 配置；不要把本页静态 HTML 的最简配置直接覆盖到需要打包的项目。
