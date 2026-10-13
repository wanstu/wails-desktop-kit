(function (global) {
  "use strict";

  const QUERY = "(prefers-color-scheme: dark)";
  const VALID = new Set(["light", "dark", "system"]);
  const PACK_PATTERN = /^[a-z0-9][a-z0-9_-]{0,63}$/;
  const PACK_LINK_ID = "desktopkit-theme-pack-stylesheet";
  let mode = "light";
  let media = null;
  let catalog = null;
  let gradientPacks = [];
  let gradientRevision = 0;

  function setResolvedTheme(theme) {
    const root = document.documentElement;
    root.setAttribute("data-dk-theme", theme);
    root.setAttribute("data-dk-theme-mode", mode);
    if (gradientPacks.length) paintGradient();
    return theme;
  }

  function resolveSystem() {
    return media && media.matches ? "dark" : "light";
  }

  function onSystemChange() {
    if (mode === "system") {
      setResolvedTheme(resolveSystem());
    }
  }

  function detachSystemListener() {
    if (!media) return;
    if (typeof media.removeEventListener === "function") {
      media.removeEventListener("change", onSystemChange);
    } else if (typeof media.removeListener === "function") {
      media.removeListener(onSystemChange);
    }
    media = null;
  }

  function attachSystemListener() {
    if (typeof global.matchMedia !== "function") {
      return setResolvedTheme("light");
    }
    media = global.matchMedia(QUERY);
    if (typeof media.addEventListener === "function") {
      media.addEventListener("change", onSystemChange);
    } else if (typeof media.addListener === "function") {
      media.addListener(onSystemChange);
    }
    return setResolvedTheme(resolveSystem());
  }

  function apply(nextMode) {
    if (!VALID.has(nextMode)) {
      throw new TypeError("desktopKitTheme.apply: expected light, dark, or system");
    }
    detachSystemListener();
    mode = nextMode;
    if (mode === "system") {
      return attachSystemListener();
    }
    return setResolvedTheme(mode);
  }

  function getMode() {
    return mode;
  }

  function getResolvedTheme() {
    const value = document.documentElement.getAttribute("data-dk-theme");
    return value === "dark" ? "dark" : "light";
  }

  function validatePackName(name, method) {
    if (typeof name !== "string" || !PACK_PATTERN.test(name)) {
      throw new TypeError(method + ": expected lowercase pack name using letters, numbers, - or _");
    }
  }

  function setPack(name) {
    validatePackName(name, "desktopKitTheme.setPack");
    document.documentElement.setAttribute("data-dk-theme-pack", name);
    return name;
  }

  function getPack() {
    return document.documentElement.getAttribute("data-dk-theme-pack") || "";
  }

  function clearPack() {
    document.documentElement.removeAttribute("data-dk-theme-pack");
  }

  function currentPackInfo(name) {
    if (!catalog || !Array.isArray(catalog.packs)) return null;
    return catalog.packs.find((pack) => pack && pack.name === name) || null;
  }

  function packStylesheetPath(name) {
    const pack = currentPackInfo(name);
    const revision = pack && typeof pack.sha256 === "string" ? pack.sha256.slice(0, 16) : "";
    return "/desktopkit-theme/" + encodeURIComponent(name) + ".css" + (revision ? "?v=" + revision : "");
  }

  function ensurePackLink() {
    let link = document.getElementById(PACK_LINK_ID);
    if (link) return link;

    link = document.createElement("link");
    link.id = PACK_LINK_ID;
    link.rel = "stylesheet";
    const tokens = Array.from(document.querySelectorAll('link[rel="stylesheet"]'))
      .find((item) => (item.getAttribute("href") || "").includes("/desktopkit/tokens.css"));
    if (tokens && tokens.parentNode) {
      tokens.insertAdjacentElement("afterend", link);
    } else {
      document.head.appendChild(link);
    }
    return link;
  }

  function applyPack(name) {
    validatePackName(name, "desktopKitTheme.applyPack");
    const link = ensurePackLink();
    const href = packStylesheetPath(name);
    setPack(name);

    if (link.getAttribute("href") === href) {
      return Promise.resolve(name);
    }

    return new Promise((resolve, reject) => {
      link.onload = () => {
        link.onload = null;
        link.onerror = null;
        resolve(name);
      };
      link.onerror = () => {
        link.onload = null;
        link.onerror = null;
        reject(new Error("Desktop Kit Theme Pack 加载失败: " + name));
      };
      link.setAttribute("href", href);
    });
  }

  function clearAppliedPack() {
    clearPack();
    const link = document.getElementById(PACK_LINK_ID);
    if (link) link.remove();
  }

  async function loadCatalog(options) {
    const refresh = Boolean(options && options.refresh);
    const response = await fetch("/desktopkit-theme/manifest.json" + (refresh ? "?refresh=1" : ""), {
      cache: "no-store"
    });
    if (!response.ok) {
      throw new Error("Desktop Kit Theme catalog 加载失败: HTTP " + response.status);
    }
    const next = await response.json();
    if (!next || !Array.isArray(next.packs)) {
      throw new Error("Desktop Kit Theme catalog 格式无效");
    }
    catalog = next;
    if (gradientPacks.length) paintGradient();
    return next;
  }

  function refreshCatalog() {
    return loadCatalog({ refresh: true });
  }

  function getCatalog() {
    return catalog;
  }


  const GRADIENT_TOKENS = [
    "--dk-app-gradient", "--dk-sidebar-gradient", "--dk-primary-gradient",
    "--dk-nav-gradient", "--dk-primary-contrast", "--dk-panel-gradient",
    "--dk-bg-app", "--dk-bg-elevated", "--dk-bg-subtle", "--dk-bg-muted",
    "--dk-text", "--dk-text-secondary", "--dk-border"
  ];
  const HEX_COLOR = /^#[0-9a-f]{6}$/i;

  function gradientColor(pack, dark) {
    const value = dark ? pack.gradient_dark : pack.gradient_light;
    if (typeof value !== "string" || !HEX_COLOR.test(value)) {
      throw new Error("主题 " + pack.name + " 缺少有效渐变色，请更新 Theme Catalog");
    }
    return value;
  }

  function colorChannels(hex) {
    return [1, 3, 5].map((position) => parseInt(hex.slice(position, position + 2), 16));
  }

  function rgba(hex, alpha) {
    return "rgba(" + colorChannels(hex).join(", ") + ", " + alpha + ")";
  }

  function darken(hex, ratio) {
    return "#" + colorChannels(hex).map((channel) =>
      Math.round(channel * ratio).toString(16).padStart(2, "0")
    ).join("");
  }

  function averageColors(colors) {
    const sums = [0, 0, 0];
    colors.forEach((color) => {
      colorChannels(color).forEach((channel, index) => { sums[index] += channel; });
    });
    return "#" + sums.map((total) => Math.round(total / colors.length).toString(16).padStart(2, "0")).join("");
  }

  function whiteContrast(hex) {
    const linear = colorChannels(hex).map((channel) => {
      const value = channel / 255;
      return value <= 0.04045 ? value / 12.92 : Math.pow((value + 0.055) / 1.055, 2.4);
    });
    const luminosity = linear[0] * .2126 + linear[1] * .7152 + linear[2] * .0722;
    return 1.05 / (luminosity + .05);
  }

  function accessibleGradientColor(hex) {
    // Gradient buttons use white text. All stops must remain legible in both modes.
    let candidate = hex;
    while (whiteContrast(candidate) < 4.8) {
      candidate = darken(candidate, .94);
    }
    return candidate;
  }

  function paintGradientSurfaces(root, dark) {
    const keys = ["app", "elevated", "subtle", "muted", "text", "text_secondary", "border"];
    const surfaces = gradientPacks.map((name) => {
      const pack = currentPackInfo(name);
      return pack && (dark ? pack.gradient_surfaces_dark : pack.gradient_surfaces_light);
    });
    keys.forEach((key) => root.style.removeProperty("--dk-" + ({
      app: "bg-app", elevated: "bg-elevated", subtle: "bg-subtle", muted: "bg-muted",
      text: "text", text_secondary: "text-secondary", border: "border"
    })[key]));
    if (!surfaces.every((surface) =>
      surface && keys.every((key) => HEX_COLOR.test(surface[key])))) return false;
    const tokens = {
      app: "--dk-bg-app", elevated: "--dk-bg-elevated",
      subtle: "--dk-bg-subtle", muted: "--dk-bg-muted",
      text: "--dk-text", text_secondary: "--dk-text-secondary", border: "--dk-border"
    };
    keys.forEach((key) => root.style.setProperty(tokens[key],
      averageColors(surfaces.map((surface) => surface[key]))));
    return true;
  }

  function gradientStops(colors, transform) {
    return colors.map((color, index) => {
      const percent = Math.round(index * 100 / (colors.length - 1));
      return transform(color) + " " + percent + "%";
    }).join(", ");
  }

  function paintGradient() {
    if (!gradientPacks.length || !catalog || !Array.isArray(catalog.packs)) return;
    const dark = getResolvedTheme() === "dark";
    const colors = gradientPacks.flatMap((name) => {
      const pack = currentPackInfo(name);
      if (!pack) throw new Error("渐变所用主题不存在: " + name);
      const primary = gradientColor(pack, dark);
      const secondary = dark ? pack.gradient_dark_secondary : pack.gradient_light_secondary;
      if (secondary && HEX_COLOR.test(secondary) && secondary.toLowerCase() !== primary.toLowerCase()) {
        return [primary, secondary];
      }
      return [primary];
    });
    const root = document.documentElement;
    const hasSurfaceComposition = paintGradientSurfaces(root, dark);
    if (hasSurfaceComposition) root.style.setProperty("--dk-panel-gradient",
      "linear-gradient(135deg, " + gradientStops(colors, (c) => rgba(c, dark ? .045 : .035)) + ")");
    else root.style.removeProperty("--dk-panel-gradient");
    root.style.setProperty("--dk-app-gradient",
      "linear-gradient(125deg, " + gradientStops(colors, (c) => rgba(c, dark ? .14 : .15)) + ")");
    root.style.setProperty("--dk-sidebar-gradient",
      "linear-gradient(155deg, " + gradientStops(colors, (c) => rgba(c, dark ? .11 : .08)) + ")");
    // Buttons are compact controls, not a full palette preview: use two primary colors.
    // Keep the complete multi-color palette on backgrounds, cards and navigation.
    const buttonColors = [gradientPacks[0], gradientPacks[gradientPacks.length - 1]]
      .map((name) => gradientColor(currentPackInfo(name), dark));
    root.style.setProperty("--dk-primary-gradient",
      "linear-gradient(115deg, " + gradientStops(buttonColors, (c) => accessibleGradientColor(darken(c, dark ? .64 : .74))) + ")");
    root.style.setProperty("--dk-nav-gradient",
      "linear-gradient(100deg, " + gradientStops(colors, (c) => rgba(c, dark ? .22 : .14)) + ")");
    root.style.setProperty("--dk-primary-contrast", "#ffffff");
    root.setAttribute("data-dk-theme-style", "gradient");
  }

  async function applyGradient(names) {
    if (!Array.isArray(names) || names.length < 2 || names.length > 4 ||
        new Set(names).size !== names.length) {
      throw new TypeError("desktopKitTheme.applyGradient: expected 2-4 distinct theme names");
    }
    names.forEach((name) => validatePackName(name, "desktopKitTheme.applyGradient"));
    const request = ++gradientRevision;
    if (!catalog) await loadCatalog();
    // Validate every color before changing the active theme.
    names.forEach((name) => {
      const pack = currentPackInfo(name);
      if (!pack) throw new Error("未知渐变主题: " + name);
      gradientColor(pack, false);
      gradientColor(pack, true);
      for (const color of [pack.gradient_light_secondary, pack.gradient_dark_secondary]) {
        if (color && !HEX_COLOR.test(color)) throw new Error("主题 " + name + " 的辅助渐变色无效");
      }
    });
    if (request !== gradientRevision) return getGradientPacks();
    await applyPack(names[0]);
    if (request !== gradientRevision) return getGradientPacks();
    gradientPacks = names.slice();
    paintGradient();
    return getGradientPacks();
  }

  function clearGradient() {
    gradientRevision++;
    gradientPacks = [];
    const root = document.documentElement;
    root.removeAttribute("data-dk-theme-style");
    GRADIENT_TOKENS.forEach((name) => root.style.removeProperty(name));
  }

  function getGradientPacks() {
    return gradientPacks.slice();
  }

  function getStyle() {
    return gradientPacks.length ? "gradient" : "solid";
  }

  function dispose() {
    detachSystemListener();
  }

  global.desktopKitTheme = Object.freeze({
    apply,
    getMode,
    getResolvedTheme,
    setPack,
    getPack,
    clearPack,
    applyPack,
    clearAppliedPack,
    loadCatalog,
    refreshCatalog,
    getCatalog,
    applyGradient,
    clearGradient,
    getGradientPacks,
    getStyle,
    dispose
  });

  if (!document.querySelector('script[data-desktopkit-about]')) {
    const aboutScript = document.createElement("script");
    aboutScript.src = "/desktopkit/about.js";
    aboutScript.defer = true;
    aboutScript.dataset.desktopkitAbout = "1";
    document.head.appendChild(aboutScript);
  }
})(window);
