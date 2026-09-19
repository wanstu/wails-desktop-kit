# 构建、CI 与 Release

[返回首页](../README.md) · [快速接入](getting-started.md) · [进度与待办](status.md)

本文的 YAML 固定当前版本线 v0.8.0。Go Module 和 reusable workflow 是两个独立版本引用，升级时应同时检查；Go 依赖升级不会自动升级 workflow。

## 本地构建

在含 `wails.json` 的目录执行：

| 平台 | 命令 | 默认产物 |
| --- | --- | --- |
| Windows | `wails build -clean` | `build/bin/<app-name>.exe` |
| Linux | `wails build -clean -tags webkit2_41` | `build/bin/<app-name>` |
| macOS | `wails build -clean -platform darwin/universal` | `build/bin/<app-name>.app` |

Linux 构建依赖可在 Debian／Ubuntu 安装：

~~~bash
sudo apt-get update
sudo apt-get install -y build-essential libgtk-3-dev libwebkit2gtk-4.1-dev
~~~

实际运行还需要图形会话；容器中的编译通过不代表托盘宿主存在。macOS 使用 Wails 构建器处理框架链接和 .app 打包，不能把裸 `go run` 的结果视为标准打包验收。

## Packaging Pipeline

v0.6.0 开始把“构建”和“打包”分开。Wails 负责生成平台程序，Desktop Kit 的 Packaging Pipeline 负责把程序整理成 Release 资产。

Linux 当前内置 `raw`、`deb`、`tar.gz` 三种格式，并为每个产物自动生成同名 `.sha256`。本地也可以直接使用：

~~~powershell
go install github.com/wanstu/wails-desktop-kit/cmd/desktopkit@v0.8.0

desktopkit package linux `
  --input .\build\bin\desktop-demo `
  --dist .\dist `
  --app-name desktop-demo `
  --asset-base desktop-demo-v1.2.0 `
  --package-version 1.2.0 `
  --formats raw,deb,tar.gz `
  --description "Desktop Demo" `
  --maintainer "wanstu" `
  --desktop-file .\packaging\desktop-demo.desktop `
  --icon .\build\appicon.png
~~~

`.deb` 可把程序安装到 `/usr/bin/<app-name>`，并在提供资源时同时安装 `.desktop` 与 hicolor 图标。`--extra-bin name=path` 可重复提供，把 CLI/helper 一并安装到 `/usr/bin`；`.tar.gz` 同样包含这些额外程序。`raw` 格式仍只代表主程序。

Packaging API 以 format builder 为边界。以后增加 AppImage 等格式时可以新增 builder，而不需要改变产品的 Wails 构建逻辑。

## CI caller

保存为业务仓库 `.github/workflows/ci.yml`。下面假设应用的 `wails.json` 在仓库根目录，`outputfilename` 为 `desktop-demo`：

~~~yaml
name: CI

on:
  push:
    branches: [master, main]
  pull_request:
    branches: [master, main]
  workflow_dispatch:

permissions:
  contents: read

jobs:
  desktop:
    uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.8.0
    with:
      app-name: desktop-demo
      desktop-dir: .
~~~

默认并行构建 Windows amd64、Linux amd64、macOS universal。v0.8.0 起可通过 `build-windows`、`build-linux`、`build-macos` 关闭不支持的平台；matrix 只创建实际启用的平台 job。单个平台失败不会立即取消其他平台；所有启用平台必须成功后才可能进入 Release。

如果应用位于子目录，将 `desktop-dir` 改为相对仓库根目录的路径，例如 `cmd/frp-client-desktop`。该目录用于直接 Wails 构建和产物定位；默认 Test／Vet 和产品 wrapper 均从仓库根目录执行。

## Release caller

保存为业务仓库 `.github/workflows/release.yml`：

~~~yaml
name: Release

on:
  push:
    tags: ["v*"]

permissions:
  contents: write

jobs:
  release:
    uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.8.0
    with:
      app-name: desktop-demo
      desktop-dir: .
      linux-package-formats: raw,deb,tar.gz
      linux-package-description: Desktop Demo
      linux-desktop-file: packaging/desktop-demo.desktop
      linux-icon-file: build/appicon.png
      publish-release: true
~~~

发布需同时满足：caller 给出写权限、`publish-release: true`、当前 ref 是 `refs/tags/v...`，且三平台构建成功。普通分支／PR CI 的 Release 跳过是正常结果，不是发布路径已验证。

带连字符的 tag 会标记为 prerelease，例如 `v1.3.0-rc.1`；普通 `v1.3.0` 不会。先在适当的候选版本验证真实 tag 发布，再决定正式版本。

构建 job 显式降为 `contents: read`；发布 job 继承 caller 权限，Kit 不自行提升 token 权限。这样普通只读 CI 也能调用同一工作流。

## 支持的输入

| 输入 | 默认值 | 说明 |
| --- | --- | --- |
| `app-name` | 必填 | 必须等于 wails.json 的 outputfilename，不带扩展名 |
| `desktop-dir` | `.` | 含 wails.json 的目录 |
| `go-version-file` | `go.mod` | 相对仓库根目录；用于选择 Go 版本 |
| `node-version` | `24` | 前端构建使用的 Node.js 版本 |
| `wails-version` | `v2.15.0` | 安装的 Wails CLI 版本 |
| `desktopkit-cli-version` | `v0.8.0` | Packaging helper 版本 |
| `build-windows` / `build-linux` / `build-macos` | `true` | 选择实际构建的平台；至少启用一个 |
| `test-command` | `go test ./...` | 设为空字符串可跳过这个公共步骤 |
| `vet-command` | `go vet ./...` | 同上 |
| `build-command-windows` | 空 | Windows 产品 wrapper |
| `build-command-linux` / `build-command-macos` | 空 | 平台专属 wrapper，优先于旧 `build-command-unix` |
| `build-command-unix` | 空 | 兼容旧 caller 的 Linux/macOS 共用 wrapper |
| `linux-package-formats` | `raw` | 逗号分隔：`raw,deb,tar.gz` |
| `linux-package-name` | 空 | Debian 包名；空时由 app-name 推导 |
| `linux-package-description` | `Wails desktop application` | Linux 包描述 |
| `linux-package-maintainer` | 仓库 owner | Debian Maintainer |
| `linux-deb-depends` | GTK3 + WebKitGTK 4.1 | Debian Depends |
| `linux-deb-section` | `utils` | Debian Section |
| `linux-deb-priority` | `optional` | Debian Priority |
| `linux-desktop-file` | 空 | 可选 `.desktop` 文件 |
| `linux-icon-file` | 空 | 可选图标文件 |
| `linux-extra-bins` | 空 | 换行分隔的 `name=path`；把 CLI/helper 一并安装进 deb/tar.gz |
| `post-package-command-windows` | 空 | 向 `dist/` 增加额外 Windows 产物 |
| `post-package-command-linux` | 空 | 向 `dist/` 增加额外 Linux 产物，例如 AppImage |
| `post-package-command-macos` | 空 | 向 `dist/` 增加额外 macOS 产物 |
| `linux-deb` | `false` | 兼容旧 caller；true 等价于追加 `deb` |
| `linux-deb-description` | 空 | 兼容旧 caller；非空时覆盖新 Description |
| `publish-release` | `false` | 是否允许进入 tag Release 发布 |

`linux-deb` 与 `linux-deb-description` 进入兼容模式，新项目优先使用 `linux-package-formats` 与 `linux-package-description`。v0.5.2 已热修旧 `.deb` control 换行；已有 caller 继续兼容。v0.8.0 起平台集合由三个 `build-*` 输入决定，Windows-only 或 Linux-only 产品不再需要复制整套 workflow。

### 更复杂格式的扩展点

复杂打包不应该继续往 reusable workflow 里堆平台专用 shell。v0.6.0 提供两层扩展：内置 format builder 负责稳定格式；product post-package hook 负责尚未升为一等能力的复杂格式。

例如 AppImage 可以先由产品脚本生成到 `dist/`：

~~~yaml
      post-package-command-linux: bash ./scripts/package-appimage.sh
~~~

hook 完成后 workflow 会扫描 `dist/`，为所有非 `.sha256` 文件生成 SHA256；Release job 也不再写死 `.deb` 或 `.tar.gz` 文件名，而是校验本次构建实际产生的全部 checksum。未来把 AppImage 升级成 Kit 内置 builder 时，不需要重构 Release 汇总阶段。

## 保留产品构建脚本

FRP 一类产品在 `with` 中追加：

~~~yaml
      test-command: ""
      vet-command: ""
      build-command-windows: ./scripts/build.ps1
      build-command-unix: bash ./scripts/build.sh
~~~

只有 wrapper 已经包含测试／vet 时才置空，避免遗漏检查。wrapper 需负责前端校验、资源准备和构建，并将最终产物放到上表约定位置。

Windows 外部命令失败必须显式传播退出码，单靠 `$ErrorActionPreference = 'Stop'` 不够：

~~~powershell
$ErrorActionPreference = 'Stop'
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go vet ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
wails build -clean
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
~~~

Unix wrapper 使用 `set -euo pipefail`，Linux 构建加 `webkit2_41`，macOS 构建加 `-platform darwin/universal`。如果测试本身引入 Wails 的 Linux WebView/assetserver，也需给对应测试／vet 命令设置 WebKit 4.1 tag。

command 输入就是可执行脚本，只应来自可信仓库配置。不要把 PR 标题等外部文本拼接到 command 输入中。

## 产物与校验

每个平台上传一个 artifact，保留 14 天。artifact 容器名稳定；**tag 构建时，文件名自动带 tag 版本号**：

| artifact 名称 | 典型 Release 文件 |
| --- | --- |
| `<app-name>-windows-amd64` | `<app-name>-v1.3.0-windows-amd64.exe` |
| `<app-name>-linux-amd64` | 由 `linux-package-formats` 决定 raw / deb / tar.gz |
| `<app-name>-macos-universal` | `<app-name>-v1.3.0-macos-universal.app.zip` |

例如 Linux 设置 `raw,deb,tar.gz` 后会得到：

~~~text
desktop-demo-v1.3.0-linux-amd64
desktop-demo-v1.3.0-linux-amd64.deb
desktop-demo-v1.3.0-linux-amd64.tar.gz
~~~

以及每个文件对应的 `.sha256`。post-package hook 新增到 `dist/` 的文件同样会自动生成 checksum 并进入 GitHub Release。

Release job 不再写死 package 格式，而是验证下载到的全部 `.sha256`，并拒绝空文件。SHA256 用于完整性校验，不是代码签名。当前仍未内置 Windows Authenticode、macOS Developer ID/notarization、AppImage、DMG 或 MSI；这些可以先通过 post-package hook 使用，后续再逐步升为 Kit 内置 builder。

## 消费者版本治理

v0.8.0 的 CLI 可以直接检查 Go Module、reusable workflow、Wails module 和 `wails.json` 的版本/命名漂移：

~~~powershell
desktopkit doctor --root .
~~~

发现 Kit module 与 workflow 版本不一致时返回非零退出码，适合本地升级前或 CI 检查。升级时可同时修改 `go.mod` 和所有 reusable workflow 引用：

~~~powershell
desktopkit upgrade --root . --to v0.8.0
~~~

默认随后执行 `go mod tidy`；只做文本升级可加 `--tidy=false`。升级命令不会改业务源码，也不会自动改变 Runtime policy。

## 常见问题

| 现象 | 检查项 |
| --- | --- |
| 找不到产物 | app-name、outputfilename、desktop-dir 与 wrapper 输出路径是否一致 |
| Release 被跳过 | 是否同时是 v 前缀 tag 且 publish-release=true |
| token 权限不足 | 写权限应由 Release caller 提供；普通 CI 保持只读 |
| Linux 报 webkit2gtk-4.0 缺失 | 安装 4.1 依赖，并在受影响的构建／测试步骤设置 webkit2_41 |
| macOS 裸 go run 缺少 UTType 符号 | 优先用 Wails 构建器；它会补齐 UniformTypeIdentifiers 框架链接 |
| wrapper 报错但 CI 看似成功 | 检查 PowerShell 的 LASTEXITCODE 是否正确退出 |

Packaging Pipeline 已通过 Kit 自身 Linux `dpkg-deb` / `tar` / SHA256 验收，并已由 Know Me v0.1.0 的真实 tag Release 验证 raw / deb / tar.gz / macOS app.zip / Windows exe 汇总发布。准确状态见 [进度清单](status.md)。
