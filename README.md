# Wails Desktop Kit

A shared desktop application foundation extracted from IME Lock v2, FRP Client
Manager, AI Dev Manager, and CodexPro+.

The goal is to keep product repositories focused on domain logic while one
versioned kit owns repeated Wails desktop behavior, cross-platform system
integration, visual language, and standard GitHub build/release engineering.

## Current scope

The first extraction provides:

- Wails application shell with shared lifecycle wiring.
- Cross-platform Wails single-instance behavior.
- Safe hide-to-tray policy.
- Declarative tray menu actions and checkboxes.
- Standard show/hide/quit tray behavior.
- Cross-platform launch-at-login:
  - Windows HKCU Run.
  - Linux XDG autostart desktop entry.
  - macOS LaunchAgent.
- Shared UI design tokens and components.
- Shared sidebar/tabs navigation styles.
- Reusable Windows/Linux/macOS GitHub Actions build workflow with checksums and
  optional tag release publishing.

The kit deliberately does not own application domain behavior such as FRP
process management, ADM Gateway/MCP logic, IME repair, or CodexPro workspace
processes.

## Install

~~~powershell
go get github.com/wanstu/wails-desktop-kit@v0.1.2
~~~

v0.1.2 is the last stable release. Runtime hardening on this branch is pending
release; consumers validating it should pin the reviewed commit and its Go
pseudo-version. Do not commit a local replace directive.

See [hardening and migration notes](docs/runtime-hardening.md) for behavior changes.

## Desktop shell example

~~~go
package main

import (
    "embed"
    "io/fs"
    "os"

    desktopkit "github.com/wanstu/wails-desktop-kit"
    "github.com/wanstu/wails-desktop-kit/autostart"
    kitui "github.com/wanstu/wails-desktop-kit/ui"
)

//go:embed all:frontend
var frontend embed.FS

//go:embed assets/appicon.png
var icon []byte

func main() {
    launch, err := desktopkit.ParseLaunchOptions(os.Args[1:])
    if err != nil {
        panic(err)
    }

    assets, err := fs.Sub(frontend, "frontend")
    if err != nil {
        panic(err)
    }

    login, err := autostart.New(autostart.Config{
        ID:          "com.wanstu.example",
        DisplayName: "Example",
        Comment:     "Example desktop utility",
        Arguments:   []string{"--autostart"},
    })
    if err != nil {
        panic(err)
    }

    window := desktopkit.DefaultWindowConfig()
    window.Width = 1080
    window.Height = 720

    err = desktopkit.Run(desktopkit.Config{
        ID:             "com.wanstu.example",
        Title:          "Example",
        Assets:         kitui.Mount(assets),
        Bind:           []interface{}{NewApp()},
        Launch:         launch,
        Window:         window,
        SingleInstance: true,
        Tray: desktopkit.TrayConfig{
            Enabled:   true,
            Icon:      icon,
            AutoStart: login,
            Items: []desktopkit.TrayItem{
                desktopkit.Action("刷新", func(c *desktopkit.Controller) error {
                    return nil
                }),
            },
        },
    })
    if err != nil {
        panic(err)
    }
}
~~~

With ui.Mount, application HTML can load shared styles directly:

~~~html
<link rel="stylesheet" href="/desktopkit/tokens.css">
<link rel="stylesheet" href="/desktopkit/base.css">
<link rel="stylesheet" href="/desktopkit/components.css">
<link rel="stylesheet" href="/desktopkit/navigation.css">
~~~

## Window safety policy

HideSafe is the default framework policy.

- Windows: hide-on-close requires confirmed tray availability.
- macOS: hide-on-close requires successful native status item creation.
- Linux: hide-on-close is not enabled automatically yet.

Linux tray availability depends on the desktop session's StatusNotifier host.
A missing tray host must not make an application window unreachable. Future
runtime tray-host detection can be implemented once in this kit without
changing consumer applications.

Applications can explicitly choose HideAlways or HideNever when required.
HideAlways still requires a running tray backend; on Linux it assumes the user
has a visible StatusNotifier host. Autostart begins visible and hides only after
readiness; a manual show cancels that pending hide. Tray failure restores the
window.

## Icon tooling

The cross-platform CLI replaces duplicated PowerShell/System.Drawing icon normalization scripts from the source projects.

Install from the repository:

~~~powershell
go install ./cmd/desktopkit
~~~

Normalize application, window, or tray artwork to a square transparent PNG:

~~~powershell
desktopkit icon --input assets/icons/app.png --output build/appicon.png --canvas 1024 --fill 0.94
~~~

By default the command trims transparent margins before fitting the visible artwork. It runs on Windows, Linux, and macOS without System.Drawing.

## Reusable GitHub workflow

A standard Wails app can call:

~~~yaml
jobs:
  desktop:
    uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.1.2
    with:
      app-name: frp-client-manager
      desktop-dir: cmd/frp-client-desktop
~~~

For a tag release:

~~~yaml
permissions:
  contents: write

jobs:
  release:
    uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.1.2
    with:
      app-name: frp-client-manager
      desktop-dir: cmd/frp-client-desktop
      publish-release: true
~~~

The workflow assumes wails.json outputfilename equals app-name. It builds
Windows amd64, Linux amd64 with webkit2_41, and macOS universal. It stages
normalized assets and SHA256 files.

Projects with their own validated build wrappers can keep them while reusing the
shared matrix and release pipeline:

~~~yaml
with:
  app-name: frp-client-manager
  desktop-dir: cmd/frp-client-desktop
  build-command-windows: ./scripts/build.ps1
  build-command-unix: bash ./scripts/build.sh
~~~

The wrapper runs from the repository root. Leave these inputs empty to use the
standard direct Wails build.

Complex products such as ADM (desktop plus multi-platform CLI) and CodexPro+
(extra core build) should keep product-specific orchestration and reuse
lower-level kit pieces instead of forcing all behavior through one oversized
workflow.

## Extraction plan

1. Stabilise the kit itself.
2. Migrate FRP Client Manager first and require behavior parity.
3. Migrate AI Dev Manager to validate extension points.
4. Refine UI components from both migrated products.
5. Migrate IME Lock v2 and CodexPro+.
6. Extend the desktopkit CLI from icon normalization to build/package helpers.

See docs/architecture.md and docs/frp-migration.md.
