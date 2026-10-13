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
const manifest = { packs: [
  { name: "aurora", sha256: "123abc", gradient_light: "#6366f1", gradient_dark: "#818cf8" },
  { name: "sakura", sha256: "456abc", gradient_light: "#d94f89", gradient_dark: "#f7a1c6" },
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
  assert.match(styles.get("--dk-primary-gradient"), /linear-gradient/);
  assert.equal(kit.getGradientPacks().length, 2);
  kit.apply("dark");
  assert.match(styles.get("--dk-app-gradient"), /rgba\(129, 140, 248, 0\.22\)/);

  kit.apply("system");
  systemDark = false;
  mediaChange();
  assert.equal(kit.getResolvedTheme(), "light");
  assert.match(styles.get("--dk-app-gradient"), /0\.15/);
  await kit.applyGradient(["sakura", "desert", "aurora"]);
  assert.equal(kit.getGradientPacks().join(","), "sakura,desert,aurora");
  assert.equal(kit.getPack(), "sakura");
  try {
    await kit.applyGradient(["aurora", "aurora"]);
    assert.fail("duplicate gradient pack names should fail");
  } catch (error) {
    assert.match(error.message, /2-4 distinct/);
  }
  kit.clearGradient();
  assert.equal(kit.getStyle(), "solid");
  assert.equal(root.style.getPropertyValue("--dk-app-gradient"), "");
  assert.equal(attrs.has("data-dk-theme-style"), false);
  console.log("gradient runtime smoke test OK");
})().catch((error) => { console.error(error); process.exitCode = 1; });
