# 组合渐变主题

Desktop Kit 的 Theme Runtime 可以组合 2～4 个基础 Theme Pack 的主色及辅助渐变色，形成统一渐变风格。原有单色主题、light/dark/system 模式保持兼容。

## API

在页面引入 Kit 的 `tokens.css`、`base.css`、`components.css`、`navigation.css` 与 `theme.js` 后：

```js
// 纯色风格（现有 API）
desktopKitTheme.apply("system");
await desktopKitTheme.applyPack("aurora");

// 由两到四个基础主题组成渐变，列表顺序就是渐变顺序
await desktopKitTheme.applyGradient(["aurora", "sakura", "ocean"]);
console.log(desktopKitTheme.getStyle());         // "gradient"
console.log(desktopKitTheme.getGradientPacks()); // ["aurora","sakura","ocean"]

// 回到纯色风格，保留当前基础主题选择
desktopKitTheme.clearGradient();
await desktopKitTheme.applyPack("forest");
```

`applyGradient` 异步读取在线或本地缓存主题清单，校验主题名称以及 light/dark 配色，加载第一个主题的 CSS 后计算组合效果。重复主题、少于 2 个或多于 4 个主题会被拒绝。

- 每个基础主题可提供浅色和深色各两种渐变锚点（`gradient_light`/`gradient_light_secondary`、`gradient_dark`/`gradient_dark_secondary`）。组合时按所选主题顺序串联所有锚点，而不是只取第一个主题的配色。旧主题缺少辅助锚点时自动退化为单个锚点。
- 选中主题均提供完整的 `gradient_surfaces_light/dark` 元数据时，运行时还会均衡合成背景、卡片、次级背景、正文及边框色，而不是让第一套主题独占界面外观；旧主题不提供时仍退回原来逻辑。
- 背景、侧栏、选中导航和主按钮按不同强度使用组合渐变，卡片也有轻量光效。渐变主按钮的所有色标按白色文字至少 4.8:1 的目标自动暗化，以确保文字易读。
- 明暗模式由 `desktopKitTheme.apply("light"|"dark"|"system")` 决定，渐变会自动响应模式变化
- `getStyle()` 返回 `"solid"` 或 `"gradient"`
- `getGradientPacks()` 返回当前组合的副本，可用于自行保存
- Kit 不持久化应用设置；具体应用负责保存/恢复模式、组合、基础主题
- 旧 Theme Manifest 仍可读取；没有 `gradient_light` 和 `gradient_dark` 的主题仅能作为纯色主题使用
- Kit 内置的四套离线 fallback 主题包含组合色值，其余在线主题需要新版 Theme Manifest

Theme Gallery 的左侧工具栏实现了可直接参考的二级选择 UI（纯色/渐变、16 套精选和全部 31 套的筛选、顺序和预设）；同屏对比页提供每套精选主题的 light/dark 真实色板模拟。原有 31 个主题名称与 ID 均保留。
