(function (global) {
  "use strict";

  const QUERY = "(prefers-color-scheme: dark)";
  const VALID = new Set(["light", "dark", "system"]);
  const PACK_PATTERN = /^[a-z0-9][a-z0-9_-]{0,63}$/;
  const PACK_LINK_ID = "desktopkit-theme-pack-stylesheet";
  let mode = "light";
  let media = null;
  let catalog = null;

  function setResolvedTheme(theme) {
    const root = document.documentElement;
    root.setAttribute("data-dk-theme", theme);
    root.setAttribute("data-dk-theme-mode", mode);
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
    return next;
  }

  function refreshCatalog() {
    return loadCatalog({ refresh: true });
  }

  function getCatalog() {
    return catalog;
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
    dispose
  });
})(window);
