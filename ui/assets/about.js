(function (global) {
  "use strict";

  let buildInfoPromise = null;

  function fallbackInfo() {
    return { version: "dev", product_version: "0.0.0", commit: "" };
  }

  async function loadBuildInfo() {
    if (!buildInfoPromise) {
      buildInfoPromise = fetch("/desktopkit-build-info.json", { cache: "no-store" })
        .then((response) => {
          if (!response.ok) return fallbackInfo();
          return response.json();
        })
        .then((value) => ({
          version: String(value && value.version || "dev"),
          product_version: String(value && value.product_version || "0.0.0"),
          commit: String(value && value.commit || "")
        }))
        .catch(() => fallbackInfo());
    }
    return buildInfoPromise;
  }

  function text(tag, className, value) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    node.textContent = value;
    return node;
  }

  class DesktopKitAbout extends HTMLElement {
    connectedCallback() {
      this.classList.add("dk-about");
      this.render(fallbackInfo());
      loadBuildInfo().then((info) => {
        if (this.isConnected) this.render(info);
      });
    }

    render(info) {
      const appName = this.getAttribute("app-name") || document.title || "Application";
      const commit = info.commit ? info.commit.slice(0, 12) : "";
      const header = document.createElement("div");
      header.className = "dk-about-heading";
      header.append(
        text("strong", "dk-about-name", appName),
        text("span", "dk-about-version", info.version)
      );

      const meta = document.createElement("div");
      meta.className = "dk-about-meta";
      if (commit) meta.append(text("span", "", "Commit " + commit));
      if (info.product_version && info.product_version !== "0.0.0") {
        meta.append(text("span", "", "Product " + info.product_version));
      }

      this.replaceChildren(header, meta);
    }
  }

  if (global.customElements && !global.customElements.get("dk-about")) {
    global.customElements.define("dk-about", DesktopKitAbout);
  }

  const kit = global.DesktopKit || {};
  kit.buildInfo = Object.freeze({ load: loadBuildInfo });
  global.DesktopKit = kit;
})(window);
