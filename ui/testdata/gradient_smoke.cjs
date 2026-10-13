const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");
const assert = require("node:assert/strict");

const attrs = new Map();
const styles = new Map();
const links = new Map();
const root = {
  setAttribute(name, value) { attrs.set(name, value); },
  getAttribute(name) { return attrs.get(name) ?? null; },
  removeAttribute(name) { attrs.delete(name); },
  style: {
    setProperty(name, value) { styles.set(name, value); },
    getPropertyValue(name) { return styles.get(name) ?? ""; },
    removeProperty(name) { styles.delete(name); }
  }
};
function link() {
  const values = new Map();
  const element = {
    id: "",
    rel: "",
    setAttribute(name, value) {
      values.set(name, value);
      if (name === "href" && this.onload) this.onload();
    },
    getAttribute(name) { return values.get(name); },
    remove() { links.delete(this.id); }
  };
  return element;
}
const document = {
  documentElement: root,
  head: { appendChild(el) { links.set(el.id, el); } },
  createElement() { return link(); },
  getElementById(id) { return links.get(id) ?? null; },
  querySelectorAll() { return []; },
  querySelector() { return {}; }
};
let systemDark = false;
let mediaChange = null;
const palettes = {
  aurora: {
    light: {app:"#e0eafb", elevated:"#ffffff", subtle:"#eef3fc", muted:"#dce6f4", text:"#182a48", text_secondary:"#52647b", border:"#bacce2"},
    dark: {app:"#080f23", elevated:"#111a33", subtle:"#192644", muted:"#253657", text:"#ebf4ff", text_secondary:"#aabedc", border:"#354a68"}
  },
  sakura: {
    light: {app:"#fff9fb", elevated:"#ffffff", subtle:"#fff0f5", muted:"#ffe7ef", text:"#51263d", text_secondary:"#946679", border:"#f3cbd9"},
    dark: {app:"#1d111c", elevated:"#291927", subtle:"#352134", muted:"#462b43", text:"#fff0f7", text_secondary:"#d8a6bf", border:"#64415b"}
  }
};
const manifest = { packs: [
  { name: "aurora", sha256: "123abc", gradient_light: "#6366f1", gradient_dark: "#818cf8", gradient_light_secondary: "#15A6A0", gradient_dark_secondary: "#B987F5", gradient_surfaces_light: palettes.aurora.light, gradient_surfaces_dark: palettes.aurora.dark },
  { name: "sakura", sha256: "456abc", gradient_light: "#d94f89", gradient_dark: "#f7a1c6", gradient_surfaces_light: palettes.sakura.light, gradient_surfaces_dark: palettes.sakura.dark },
  { name: "desert", sha256: "678abc", gradient_light: "#b66d33", gradient_dark: "#f2b56f" }
]};
const window = {
  matchMedia() {
    return {
      get matches() { return systemDark; },
      addEventListener(_, cb) { mediaChange = cb; },
      removeEventListener(_, cb) { if (mediaChange === cb) mediaChange = null; }
    };
  }
};
const sandbox = {
  document, window,
  fetch: async () => ({ ok: true, json: async () => manifest })
};
vm.runInNewContext(fs.readFileSync(path.join(__dirname, "..", "assets", "theme.js"), "utf8"), sandbox);
const kit = window.desktopKitTheme;

(async () => {
  assert.equal(kit.getStyle(), "solid");
  kit.apply("light");
  await kit.applyGradient(["aurora", "sakura"]);
  assert.equal(kit.getStyle(), "gradient");
  assert.equal(attrs.get("data-dk-theme-style"), "gradient");
  assert.match(styles.get("--dk-app-gradient"), /rgba\(99, 102, 241, 0\.15\)/);
  assert.match(styles.get("--dk-app-gradient"), /rgba\(21, 166, 160, 0\.15\)/);
  assert.match(styles.get("--dk-primary-gradient"), /linear-gradient/);
  assert.ok(styles.get("--dk-bg-app").startsWith("#"), "surface colors are composed");
  assert.ok(styles.get("--dk-panel-gradient").includes("linear-gradient"), "surface cards use gradient accents");
  assert.equal(kit.getGradientPacks().length, 2);
  kit.apply("dark");
  assert.match(styles.get("--dk-app-gradient"), /rgba\(129, 140, 248, 0\.14\)/);
  assert.match(styles.get("--dk-app-gradient"), /rgba\(185, 135, 245, 0\.14\)/);
  assert.ok(styles.get("--dk-bg-app").startsWith("#"), "dark surfaces are composed");
  const stops = styles.get("--dk-primary-gradient").match(/#[a-f0-9]{6}/gi);
  for (const stop of stops) {
    const rgb = [1,3,5].map((i) => parseInt(stop.slice(i,i+2),16) / 255);
    const linear = rgb.map((v) => v <= .04045 ? v / 12.92 : Math.pow((v+.055)/1.055,2.4));
    const contrast = 1.05 / (.2126*linear[0]+.7152*linear[1]+.0722*linear[2]+.05);
    assert.ok(contrast >= 4.5, "white button text must be readable against " + stop + ": " + contrast);
  }

  kit.apply("system");
  systemDark = false;
  mediaChange();
  assert.equal(kit.getResolvedTheme(), "light");
  assert.match(styles.get("--dk-app-gradient"), /0\.15/);
  await kit.applyGradient(["sakura", "desert", "aurora"]);
  assert.equal(kit.getGradientPacks().join(","), "sakura,desert,aurora");
  assert.equal(kit.getPack(), "sakura");
  assert.equal(styles.get("--dk-bg-app"), undefined, "legacy palette in combination should not reuse stale mixed surfaces");
  try {
    await kit.applyGradient(["aurora", "aurora"]);
    assert.fail("duplicate gradient pack names should fail");
  } catch (error) {
    assert.match(error.message, /2-4 distinct/);
  }
  kit.clearGradient();
  assert.equal(kit.getStyle(), "solid");
  assert.equal(root.style.getPropertyValue("--dk-app-gradient"), "");
  assert.equal(root.style.getPropertyValue("--dk-bg-app"), "");
  assert.equal(attrs.has("data-dk-theme-style"), false);
  console.log("gradient runtime smoke test OK");
})().catch((error) => { console.error(error); process.exitCode = 1; });
