# Runtime hardening and consumer migration

This work follows v0.1.2. It is not included in that stable tag.

## Runtime contract

- Wails owns application startup, its main thread, and shutdown. The macOS
  backend creates/removes NSStatusItem objects on the AppKit main thread; it
  does not run or stop NSApplication and does not change activation policy.
- Startup installs the controller context before the application's Startup hook.
  DomReady runs the application's hook before starting the tray. Shutdown stops
  tray work, disables the controller, and then runs the application's hook.
- BeforeClose runs for a window close and an explicit quit; returning true
  cancels either and resets quit intent. Use Controller.Quit for tray exit
  actions, especially when close-to-tray is enabled.
- Windows confirms a shell icon rectangle. macOS confirms status item creation.
  Linux confirms registration of this process's StatusNotifierItem on D-Bus.
  Linux registration does not prove a visible host; HideSafe keeps its window
  visible. Host detection remains future work.
- StartHiddenOnAutoStart is deferred until tray readiness. This may briefly
  show the window. A manual show cancels the pending hide. Failure restores
  the window before Hooks.TrayError is called (or the default error dialog).
- Native window calls execute without the controller lock. Versioned visibility
  requests restore the latest intent when calls complete out of order.
- Tray actions and checkbox state readers run serially on a bounded worker;
  repeated clicks on an already pending action are dropped. Show/hide remain
  separate from slow business actions. Shutdown stops accepting new work but
  cannot forcibly interrupt an application callback already running. Callbacks
  must terminate and should honor Controller.Context cancellation.
- SecondInstance receives the full Wails SecondInstanceData. Applications can
  ignore duplicate --autostart launches or handle files/URLs. Its default still
  shows and unminimises the window.
- TrayConfig.AutoStart accepts AutoStartProvider (Supported, Enabled,
  SetEnabled). Existing *autostart.Manager values continue to work; a product
  adapter can also publish its own state-change event after SetEnabled.

## Autostart

Linux Exec values now apply argument quoting and desktop-string escaping as
two separate steps, including percent field codes. A gio launch integration
test exercises the actual desktop parser. Multi-line commands and relative
XDG_CONFIG_HOME are rejected.

macOS writes/removes the per-user LaunchAgent for the next login. It does not
bootstrap an agent now or stop the current application when disabled. Enabled
compares the installed definition to the expected executable and arguments.
Moving the binary requires enabling autostart again at its new location; the
kit does not silently rewrite the user's setting during normal startup.

## Assets and icons

Mount must receive an application FS rooted at index.html; use fs.Sub before
Mount when embedding frontend/dist. The desktopkit directory is reserved.
The overlay implements directory enumeration and rejects invalid fs paths.
Tests cover fstest.TestFS and real Wails asset routing for root/subdirectory
embeds.

Icon inputs are checked with DecodeConfig before decoding: at most 16 million
pixels, a 4096-pixel output side, and finite fill values. Transparent edges use
premultiplied interpolation. Output is encoded to a temporary file in the
destination directory, then renamed after successful completion. An error
keeps the existing output intact, including when input and output are equal.

## Workflow boundary

Build jobs explicitly use contents: read. Release inherits the caller's
permissions; only a release caller needs contents: write. Publication requires
publish-release: true and a v-prefixed tag.

Release downloads only the three named platform artifacts from its own run,
then requires their binaries/archives and verifies all SHA256 files. Command
inputs are executable scripts supplied by a trusted caller repository; this
workflow is not a sandbox for untrusted commands.

## Verification and limits

Local checks during implementation:

- Windows go test -race ./... and go vet ./... passed.
- Debian 13 go test -tags webkit2_41 -race ./... passed, including gio launch.
- Windows production Wails/WebView2 plus tray startup/shutdown smoke passed.
- First runtime commit 3056f58 passed Windows/Linux/macOS CI (35369309819).

The updated CI also runs race tests and a macOS production Wails/tray smoke.
A build or automated smoke does not replace manual tray clicks, desktop-session
host loss, login, or an actual consumer tag release. Track those results
separately from compilation. FRP migration should preserve both exit actions,
use HideSafe, and pin both its module and reusable workflow to the same reviewed
kit revision. UI and icon-script migration are separate tasks.
