# UI 与图标

[返回首页](../README.md) · [快速接入](getting-started.md)

UI 与 Runtime 可以分别接入。Kit UI 是随 Go Module 嵌入的 CSS，不是独立 npm 组件包，也不会自动改写现有页面。

## 挂载资源

应用资源必须先对齐到含 `index.html` 的目录，再调用 `kitui.Mount`。例如构建产物位于 `frontend/dist`：

~~~go
//go:embed all:frontend/dist
var frontend embed.FS

func mountedAssets() (fs.FS, error) {
	appFS, err := fs.Sub(frontend, "frontend/dist")
	if err != nil {
		return nil, err
	}
	return kitui.Mount(appFS), nil
}
~~~

需要导入 `embed`、`io/fs` 和 `kitui "github.com/wanstu/wails-desktop-kit/ui"`，然后把结果传入 `Config.Assets` 或原有 Wails AssetServer 的 Assets。

`desktopkit/` 是保留目录，Kit 的资源优先于同名应用文件。其他路径仍由应用 FS 提供；它不会替代 Vite 开发服务器或 SPA 路由。修复版补齐了目录读取契约，并用实际 Wails asset handler 验证了资源路径。

在 HTML 按顺序加载，业务覆盖样式放最后：

~~~html
<link rel="stylesheet" href="/desktopkit/tokens.css">
<link rel="stylesheet" href="/desktopkit/base.css">
<link rel="stylesheet" href="/desktopkit/components.css">
<link rel="stylesheet" href="/desktopkit/navigation.css">
<link rel="stylesheet" href="/app.css">
~~~

## 样式使用

| 文件 | 代表类名／内容 |
| --- | --- |
| `tokens.css` | `--dk-primary`、`--dk-bg-app`、`--dk-text`、spacing、radius、shadow |
| `base.css` | `dk-app-shell`、`dk-content`、`dk-stack`、`dk-row`、基础 focus 样式 |
| `components.css` | panel/card、button、status pill、message、switch、field、dialog、log |
| `navigation.css` | `dk-sidebar-layout`、`dk-nav-group`、`dk-nav-link`、`dk-tabs`、`dk-tab` |

表单示例：

~~~html
<section class="dk-panel dk-stack">
  <div class="dk-form-grid">
    <label class="dk-field">
      名称
      <input name="name" autocomplete="off">
    </label>
    <label class="dk-field">
      模式
      <select name="mode"><option value="manual">手动</option></select>
    </label>
  </div>
  <label class="dk-setting-row">
    <strong>显示通知</strong>
    <input class="dk-switch" type="checkbox" name="notifications">
  </label>
  <div class="dk-row">
    <button class="dk-button dk-button-primary" type="button">保存</button>
    <span class="dk-status-pill is-success">已连接</span>
  </div>
</section>
~~~

input、select、textarea 放在 `dk-field` 内才会获得相应表单样式。按钮使用基础 `dk-button` 加变体；状态使用 `is-success`／`is-warning`／`is-danger`。

导航活动项设置 `aria-current="page"`；tab 可用 `aria-selected="true"` 或 `is-active` 控制样式。页面切换、键盘导航、状态更新由前端实现。原生 `dialog.dk-dialog` 仍需业务调用 `showModal()`／`close()`。

覆盖 token，例如在 `app.css`：

~~~css
:root {
  --dk-primary: #156f63;
  --dk-primary-hover: #105a50;
  --dk-sidebar-width: 240px;
}
~~~

Kit UI 提供 `light / dark / system` 三种通用主题模式，但**不负责用户设置持久化，也不决定产品默认主题**。

最基础的静态用法仍可直接声明：

~~~html
<html lang="zh-CN" data-dk-theme="dark">
~~~

需要运行时切换或跟随系统时，加载公共主题机制：

~~~html
<script src="/desktopkit/theme.js"></script>
<script>
  desktopKitTheme.apply("system");
</script>
~~~

API：

- `desktopKitTheme.apply("light")`：应用 Kit Light。
- `desktopKitTheme.apply("dark")`：应用 Kit Dark。
- `desktopKitTheme.apply("system")`：监听 `prefers-color-scheme`，系统变化时更新公共主题。
- `desktopKitTheme.getMode()`：返回应用最后显式选择的模式。
- `desktopKitTheme.getResolvedTheme()`：返回当前实际生效的 `light` 或 `dark`。
- `desktopKitTheme.setPack("midnight")`：设置可选主题包标识，只写入 `data-dk-theme-pack`，不负责加载 CSS。
- `desktopKitTheme.getPack()`：返回当前主题包名称；未设置时返回空字符串。
- `desktopKitTheme.clearPack()`：移除当前主题包标识。
- `desktopKitTheme.dispose()`：移除 system 监听，适合测试或特殊生命周期。

Kit 启动时不会主动读取 `localStorage`、配置文件或系统偏好，因此旧应用升级不会突然改变外观。应用负责保存自己的 `theme = light | dark | system`，并在启动时调用 `apply`。

公共组件颜色全部通过 `--dk-*` token 获取。主题模式和主题包是正交概念：`data-dk-theme` 只表示 light / dark 的实际明暗结果，`data-dk-theme-pack` 只表示可选视觉配色。推荐组合用法：

~~~html
<html data-dk-theme="dark" data-dk-theme-pack="midnight">
~~~

~~~js
desktopKitTheme.apply("system");
desktopKitTheme.setPack("midnight");
~~~

Kit 只维护主题协议和默认 light/dark token，不内置 Midnight、Graphite、Forest 等可选主题包。官方可选通用主题应放在独立模块 `github.com/wanstu/wails-desktop-kit-themes`；产品品牌主题仍由具体应用维护。主题包 CSS 必须只覆盖 `--dk-*` token，不重新定义 `.dk-button`、`.dk-panel`、`.dk-nav-link` 等组件规则。

自定义主题至少应覆盖背景、文字、边框、primary、状态色、导航 active、control、backdrop 与 log token，并逐页检查 hover、focus、disabled 和窄窗口布局。

如果使用独立前端开发服务器，`kitui.Mount` 不会自动给那个服务器增加路由。请配置开发资源代理或样式提供方式，并在 Wails production build 中验证最终路径。

## 图标 CLI

v0.3.1 起，`desktopkit icon` 同时提供 **generate**（确定性生成产品家族图标）和 **normalize**（规范化已有图片）两条路径。原有 `desktopkit icon --input ...` 用法保持兼容。

~~~powershell
go install github.com/wanstu/wails-desktop-kit/cmd/desktopkit@v0.4.0

# SSH / Terminal 家族图标
desktopkit icon generate --output assets/appicon.png --symbol terminal --background "#2463EB"

# FRP 这类单字母家族图标
desktopkit icon generate --output assets/appicon.png --symbol monogram --text F --background "#2463EB"

# 原有图片规范化仍然兼容
desktopkit icon --input assets/icons/source.png --output build/appicon.png --canvas 1024 --fill 0.94
~~~

也可在 Kit 源码根目录执行 `go run ./cmd/desktopkit icon ...`。

### generate 参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--output` | 必填 | PNG 输出路径 |
| `--size` | 32 | 16–1024 的正方形尺寸 |
| `--inset` | 2 | 透明外边距 |
| `--radius` | 7 | 圆角半径 |
| `--symbol` | terminal | `terminal` 或 `monogram` |
| `--text` | 空 | monogram 使用；一个 ASCII 字母或数字 |
| `--background` | #2463EB | `#RRGGBB` 或 `#RRGGBBAA` |
| `--foreground` | #FFFFFF | 前景色 |

默认生成规格与 FRP Client 的现有产品家族保持一致：透明 32×32 画布、纯色圆角底、白色几何 glyph。生成过程不调用系统字体，因此 Windows/Linux/macOS 输出一致。

对应 Go API 使用 `icon.DefaultBadgeOptions()` 与 `icon.GenerateBadgeFile(...)`。

### normalize 参数

| 参数 | 默认值 | 说明 |
| --- | --- | --- |
| `--input` | 必填 | PNG／JPEG／GIF；GIF 读取静态图像，不输出动画 |
| `--output` | 必填 | RGBA PNG 输出路径，缺失的父目录会创建 |
| `--canvas` | 1024 | 输出正方形边长，修复版最大 4096 |
| `--fill` | 0.94 | 可见图形占画布比例，通常取 (0, 1] |
| `--trim-alpha` | true | 是否先裁掉透明边缘；关闭用 `--trim-alpha=false` |
| `--alpha-threshold` | 8 | 0–255，alpha 大于该值才计入可见边界 |

Go API 的零值约定同样影响 CLI：canvas=0、fill=0 会采用默认值。负数、超出上限和非有限 fill 值会报错。全透明图片在启用裁剪时会报错。

修复版在解码前检查输入最多 **16,777,216 像素**；双线性插值处理透明边缘，输出先写同目录临时文件，成功后替换目标。失败时保留原输出，支持输入和输出同路径。

Kit 的 generate / normalize 都只负责得到稳定 PNG，不负责安装包、签名或发布。Wails 继续从 `build/appicon.png` 派生 Windows ICO、macOS ICNS 等平台资产；托盘图标仍由应用显式嵌入。产品脚本需要显式调用 CLI，安装 Kit 不会自动覆盖应用资源。

建议把生成后的 `assets/appicon.png` 提交进仓库，同时在构建脚本中再次调用 `desktopkit icon generate`，让日常 `go test` 有可嵌入资源，正式构建又能校验图标没有漂移。
