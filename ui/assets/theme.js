(function (global) {
  "use strict";

  const QUERY = "(prefers-color-scheme: dark)";
  const VALID = new Set(["light", "dark", "system"]);
  const PACK_PATTERN = /^[a-z0-9][a-z0-9_-]{0,63}$/;
  let mode = "light";
  let media = null;

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

  function setPack(name) {
    if (typeof name !== "string" || !PACK_PATTERN.test(name)) {
      throw new TypeError("desktopKitTheme.setPack: expected lowercase pack name using letters, numbers, - or _");
    }
    document.documentElement.setAttribute("data-dk-theme-pack", name);
    return name;
  }

  function getPack() {
    return document.documentElement.getAttribute("data-dk-theme-pack") || "";
  }

  function clearPack() {
    document.documentElement.removeAttribute("data-dk-theme-pack");
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
    dispose
  });
})(window);
