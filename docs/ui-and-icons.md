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

当前默认是浅色主题；部分组件还有固定颜色，不能只改一个 token 就宣称完整支持暗色主题。迁移时逐页检查控件、hover、focus、disabled 和窄窗口布局。

如果使用独立前端开发服务器，`kitui.Mount` 不会自动给那个服务器增加路由。请配置开发资源代理或样式提供方式，并在 Wails production build 中验证最终路径。

## 图标 CLI

按首页版本选择安装。下面固定本轮修复版本：

~~~powershell
go install github.com/wanstu/wails-desktop-kit/cmd/desktopkit@v0.1.3-0.20260918165411-b3145064740c
desktopkit icon --input assets/icons/source.png --output build/appicon.png --canvas 1024 --fill 0.94
~~~

也可在 Kit 源码根目录执行 `go run ./cmd/desktopkit icon ...`。

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

CLI 只规范化 PNG，不负责生成所有平台安装包、ICO／ICNS、签名或发布。Wails 的应用图标通常放 `build/appicon.png`；托盘图标由应用嵌入自己的资源文件。产品脚本需要显式调用 CLI，安装 Kit 不会自动替换旧脚本。

FRP 当前尚未将 UI 或图标脚本迁移到本页方案。
