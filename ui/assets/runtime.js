(function (global) {
  'use strict';

  function runtimeClipboard() {
    return global.runtime && typeof global.runtime === 'object' ? global.runtime : null;
  }

  async function writeText(value) {
    const text = String(value == null ? '' : value);
    const runtime = runtimeClipboard();
    if (runtime && typeof runtime.ClipboardSetText === 'function') {
      const result = await runtime.ClipboardSetText(text);
      if (result === false) throw new Error('Wails ClipboardSetText returned false');
      return true;
    }
    if (global.navigator && global.navigator.clipboard && typeof global.navigator.clipboard.writeText === 'function') {
      await global.navigator.clipboard.writeText(text);
      return true;
    }
    const document = global.document;
    if (!document || typeof document.createElement !== 'function') {
      throw new Error('Clipboard is unavailable');
    }
    const field = document.createElement('textarea');
    field.value = text;
    field.setAttribute('readonly', '');
    field.style.position = 'fixed';
    field.style.opacity = '0';
    document.body.append(field);
    field.select();
    const copied = typeof document.execCommand === 'function' && document.execCommand('copy');
    field.remove();
    if (!copied) throw new Error('Clipboard is unavailable');
    return true;
  }

  async function readText() {
    const runtime = runtimeClipboard();
    if (runtime && typeof runtime.ClipboardGetText === 'function') {
      return String(await runtime.ClipboardGetText());
    }
    if (global.navigator && global.navigator.clipboard && typeof global.navigator.clipboard.readText === 'function') {
      return String(await global.navigator.clipboard.readText());
    }
    throw new Error('Clipboard read is unavailable');
  }

  const kit = global.DesktopKit || {};
  kit.clipboard = Object.freeze({ writeText, readText });
  global.DesktopKit = kit;
})(window);
