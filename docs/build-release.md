# 构建、CI 与 Release

[返回首页](../README.md) · [快速接入](getting-started.md) · [进度与待办](status.md)

本文的 YAML 固定本轮正式版本 v0.2.0；FRP 正在切换到这一版本。继续使用稳定版时，将 Go Module 和 workflow 一起固定到 v0.1.2，并接受该版本尚无本轮修复的限制。Go 依赖升级不会自动升级 workflow。

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
    uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.2.0
    with:
      app-name: desktop-demo
      desktop-dir: .
~~~

默认并行构建 Windows amd64、Linux amd64、macOS universal。单个平台失败不会立即取消其他平台；整个构建必须成功后才可能进入 Release。

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
    uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.2.0
    with:
      app-name: desktop-demo
      desktop-dir: .
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
| `test-command` | `go test ./...` | 设为空字符串可跳过这个公共步骤 |
| `vet-command` | `go vet ./...` | 同上 |
| `build-command-windows` | 空 | 非空时使用 Windows 产品 wrapper |
| `build-command-unix` | 空 | 非空时使用 Linux/macOS 产品 wrapper |
| `publish-release` | `false` | 是否允许进入 tag Release 发布 |

目前平台矩阵固定，不提供任意矩阵、额外架构、deb 打包或签名输入。需要多 CLI、多产物图或不同平台矩阵的产品继续维护自己的编排，按需复用 Kit 包。

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

每个平台上传一个 artifact，保留 14 天。artifact 容器名保持稳定，便于 Release job 精确下载；**tag 构建时，容器内的发布文件会自动带 tag 版本号**：

| artifact 名称 | 普通分支／PR 核心文件 | tag `v1.3` 的 Release 文件 |
| --- | --- | --- |
| `<app-name>-windows-amd64` | `<app-name>-windows-amd64.exe` | `<app-name>-v1.3-windows-amd64.exe` |
| `<app-name>-linux-amd64` | `<app-name>-linux-amd64` | `<app-name>-v1.3-linux-amd64` |
| `<app-name>-macos-universal` | `<app-name>-macos-universal.app.zip` | `<app-name>-v1.3-macos-universal.app.zip` |

对应的 `.sha256` 文件使用相同基础文件名，例如 `<app-name>-v1.3-windows-amd64.exe.sha256`，文件内容中的校验目标也会同步带版本号。

发布步骤只下载同一次 run 的这三个指定 artifact，确认带当前 tag 的核心产物存在且非空，再验证 SHA256。构建端仍会上传 `dist/*`，发布端仍会附加下载目录中的文件；不要把临时文件放进 dist，也不要让各平台额外文件使用相同名称。

SHA256 用于完整性校验，不是代码签名。当前工作流没有 macOS 签名／公证，也没有 Windows 签名或 Linux 安装包抽象。

## 常见问题

| 现象 | 检查项 |
| --- | --- |
| 找不到产物 | app-name、outputfilename、desktop-dir 与 wrapper 输出路径是否一致 |
| Release 被跳过 | 是否同时是 v 前缀 tag 且 publish-release=true |
| token 权限不足 | 写权限应由 Release caller 提供；普通 CI 保持只读 |
| Linux 报 webkit2gtk-4.0 缺失 | 安装 4.1 依赖，并在受影响的构建／测试步骤设置 webkit2_41 |
| macOS 裸 go run 缺少 UTType 符号 | 优先用 Wails 构建器；它会补齐 UniformTypeIdentifiers 框架链接 |
| wrapper 报错但 CI 看似成功 | 检查 PowerShell 的 LASTEXITCODE 是否正确退出 |

本轮修复已经通过普通消费者 CI，新的消费者 tag Release 路径仍待真实发布验收。准确状态见 [进度清单](status.md)。
