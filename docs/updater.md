# Updater Core

[返回首页](../README.md) · [构建与发布](build-release.md) · [Roadmap](roadmap.md)

Updater Core 把“发现新版”和“安装新版”分开。当前阶段只提供 **Check / Download / SHA256 Verify**，不会退出应用、替换可执行文件或自动重启。

## 基础用法

~~~go
import (
    "context"
    "github.com/wanstu/wails-desktop-kit/updater"
)

client := updater.Client{
    CurrentVersion: "v1.2.3",
    Provider: updater.GitHubProvider{
        Owner:      "acme",
        Repository: "desktop-app",
    },
    SelectAsset: updater.AssetBySuffix("windows-amd64-setup.exe"),
}

check, err := client.Check(context.Background())
if err != nil {
    // 网络错误、Release 元数据异常、资产缺失或 SHA256 缺失
}
if !check.UpdateAvailable {
    // 已是最新版
}

download, err := client.Download(context.Background(), check, "./updates", func(p updater.Progress) {
    // p.Downloaded / p.Total
})
if err != nil {
    // 校验失败时不会留下最终可安装文件
}
_ = download.Path
~~~

`CurrentVersion` 和远端 tag 使用语义版本比较。支持 `v1.2.3`、`v1.2.3-rc.1` 和 build metadata；例如 `v1.2.3-rc.1 < v1.2.3`。

## Provider 边界

上层只依赖 `Provider` 接口：

~~~go
type Provider interface {
    Latest(context.Context) (Release, error)
}
~~~

当前内置 `GitHubProvider`。它读取 GitHub Releases API，忽略 draft；默认忽略 prerelease，设置 `IncludePrerelease: true` 后才进入 RC / beta 渠道。Provider 会按语义版本选择最高的可用 Release，而不是把 API 返回顺序直接当版本顺序。

自建更新源、Gitea、对象存储 manifest 等可以实现同一接口，不需要修改 `Client`。

## 资产选择

Kit 不猜业务产品的文件名。消费者显式提供选择器：

- `ExactAsset("app-v1.2.3-windows-amd64-setup.exe")`：精确文件名。
- `AssetBySuffix("windows-amd64-setup.exe")`：适合 Release 文件名前带产品名和版本号的 Kit 标准产物。

后缀选择出现多个匹配时直接报错，不会随机选一个。

## SHA256 规则

下载成功必须存在可信的 SHA256 来源，优先级：

1. Provider 元数据中资产自带的 SHA256，例如 GitHub Release asset 的 `digest: sha256:...`。
2. 同一 Release 中与目标资产同名的 `<asset>.sha256` sidecar。

两者都没有时，`Check` 返回 `ErrChecksumMissing`，不会进入可下载状态。

`Download` 先写同目录临时 `.part` 文件，一边下载一边计算 SHA256；只有大小检查和 SHA256 都通过后才重命名为最终文件。网络中断、取消或校验失败时删除临时文件，不留下“看起来可安装”的损坏产物。

SHA256 这里只提供完整性验证，不等于代码签名。Windows Authenticode / macOS signing 与 notarization 仍是独立分发安全层。

## 当前非目标

这一阶段明确不做：

- 不自动启动安装器。
- 不退出当前应用。
- 不覆盖正在运行的 Portable exe。
- 不自动重启应用。
- 不把 GitHub Token、下载目录或更新渠道写入 Kit 全局配置。

后续 Windows 自动更新会复用 v0.10.0 已验证的 NSIS Setup：只有确认当前实例为安装版、下载文件校验通过后，才进入 `Install -> Exit -> Restart`。

## 取消与超时

`Check` / `Download` 都接收 `context.Context`。产品应根据 UI 生命周期设置取消或超时；关闭更新对话框、退出应用或用户主动取消时，取消对应 context 即可。进度回调由调用方决定如何映射到 UI。
