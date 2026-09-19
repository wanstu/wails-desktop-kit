# Runtime Theme

[返回首页](../README.md)

Desktop Kit 的 Runtime Theme 让应用在**不重新构建 exe** 的情况下获得新的 Theme Pack。

## 架构

~~~text
Application
  |
  | desktopkit.Config{Theme: desktopkit.DefaultThemeConfig()}
  v
Wails Desktop Kit
  |- 内置 light / dark / system 基础 token
  |- Theme manifest client
  |- SHA-256 校验
  |- 全局本地 cache
  |- /desktopkit-theme/* 本地 AssetServer
  v
wails-desktop-kit-theme
  |- manifest.json
  `- assets/*.css
~~~

应用不需要再 import `github.com/wanstu/wails-desktop-kit-theme`，也不需要把主题 CSS go:embed 到自己的 exe。

## Go 接入

应用继续用 `ui.Mount()` 挂载 Kit 基础 UI：

~~~go
appAssets := kitui.Mount(frontend)

err := desktopkit.Run(desktopkit.Config{
    ID:     "example",
    Title:  "Example",
    Assets: appAssets,
    Theme:  desktopkit.DefaultThemeConfig(),
})
~~~

零值 `ThemeConfig` 默认关闭 Runtime Theme，因此旧应用升级 Kit 不会突然产生网络请求。

默认官方 manifest：

~~~text
https://raw.githubusercontent.com/wanstu/wails-desktop-kit-theme/master/manifest.json
~~~

需要私有源、测试源或企业镜像时可以覆盖 `ManifestURL`。

## 前端 API

加载 `/desktopkit/theme.js` 后：

~~~js
const catalog = await desktopKitTheme.loadCatalog()
const refreshed = await desktopKitTheme.refreshCatalog()
await desktopKitTheme.applyPack("aurora")
desktopKitTheme.apply("system")
~~~

`loadCatalog()` 优先使用本地 manifest cache；过期时会后台刷新。`refreshCatalog()` 会显式请求远端最新版。

应用负责保存自己的 theme mode 与 theme pack；Kit 不使用 localStorage，也不接管业务配置。

## 缓存

默认全局缓存：

~~~text
$XDG_CACHE_HOME/wails-desktop-kit/themes/
~~~

未设置 `XDG_CACHE_HOME` 时使用：

~~~text
~/.cache/wails-desktop-kit/themes/
├── manifest.json
└── packs/
    ├── aurora-<sha256>.css
    └── ocean-<sha256>.css
~~~

多个使用 Desktop Kit 的应用共享这份主题缓存，因此不会重复下载同一个 Theme Pack。

## 安全与失败行为

- 限制 manifest 与 CSS 最大尺寸。
- 校验稳定的 pack name。
- 校验每个 CSS 的 SHA-256。
- 拒绝 Theme CSS 中的 `@import` 和 `url(...)` 外部资源。
- 使用临时文件 + 原子替换写入 cache。
- CSS 请求使用 hash ETag。
- 远端 manifest 不可用时继续使用本地 last-known-good manifest。

Kit 固定内置 `aurora`（极光）、`ocean`（海洋）、`forest`（森林）、`sunset`（落日）4 个 fallback Theme Pack。即使首次启动完全离线、远端 Theme 仓库不可访问，甚至 Theme cache 目录不可写，这 4 个主题仍可列出并正常加载；基础 light/dark token 则是更底层的最终保障。业务功能不会因 Theme 服务失败而不可用。

## Theme 仓库发布契约

Theme 仓库根目录维护 `manifest.json` 与 `assets/*.css`。manifest 包含 stable name、display name、description、CSS file 和 SHA-256。

Theme 仓库新增或修改 CSS 后，需要重新生成 manifest，并在同一次提交中发布。Kit 收到新 manifest 后会先把该 manifest 引用的全部 CSS 下载并校验；只有完整成功后才替换本地 manifest。中途任意 Theme 下载失败都会继续保留上一份完整 last-known-good 快照。应用下次刷新 Theme catalog 时即可看到成功同步的变更，**无需升级 Kit，也无需重新构建应用**。

## 编译期 Theme Pack

旧的 `wails-desktop-kit-theme.MountWithKit()` 仍可用于完全离线、主题集必须固定在二进制中的场景。普通桌面应用优先使用 Runtime Theme。
