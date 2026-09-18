# Architecture

## Boundary

Wails Desktop Kit owns reusable desktop-product mechanics:

- Wails window shell and lifecycle composition.
- Window show/hide/quit controller.
- Single-instance recovery.
- Tray lifecycle and declarative menus.
- Launch-at-login backends.
- Shared UI design tokens, components, and navigation.
- Standard desktop CI/build/release conventions.

Consumer repositories own their domain:

- Processes and services managed by the product.
- Product configuration schemas and persistence.
- Product APIs and business logic.
- Product-specific release steps that are not generic Wails desktop behavior.

## Root package: desktopkit

The root package composes Wails and system integration.

Important types:

- Config
- WindowConfig
- TrayConfig
- TrayItem
- Controller
- LaunchOptions

The first API intentionally uses concrete configuration rather than a plugin
registry. These desktop utilities are small enough that explicit composition is
easier to review and less likely to turn the kit into a framework that owns
business state.

## autostart package

autostart.Manager normalizes launch-at-login around:

- Supported()
- Enabled()
- SetEnabled(bool)

Storage:

- Windows: HKCU Run value named by Config.ID.
- Linux: XDG autostart file named Config.ID.desktop.
- macOS: LaunchAgent named Config.ID.plist.

The generated entry matches the current executable path and configured
arguments. If a portable executable moves, the old entry no longer counts as
enabled until the user enables it again from the new location.

## ui package

The UI package embeds shared CSS. ui.Mount(appFS) exposes those files under the
desktopkit/ asset prefix and delegates all other files to the application.

The CSS is component-oriented rather than page-oriented. Products keep full
ownership of their screen composition and information architecture.

## icon package and CLI

The icon package owns transparent-margin trimming, square canvas fitting,
centering, and PNG generation that was previously duplicated in PowerShell
scripts. cmd/desktopkit exposes it through the cross-platform `desktopkit icon`
command.

Build and packaging CLI commands should only be added after consumer migrations
show stable inputs and extension points.

## Tray extension model

The framework owns native tray lifecycle, OS-thread pinning, standard
show/hide, launch-at-login, quit, and native error dialogs for failed actions.

Products add declarative actions and state-backed checkboxes.

A product with special exit behavior can set DisableQuit and add product
actions that perform domain cleanup and then call Controller.Quit().

## Linux safety

The kit treats Linux as unsafe for implicit hide-to-tray until runtime
StatusNotifier-host detection exists. A tray backend may be available while the
desktop session has no visible tray host. The main window therefore remains
recoverable by default.

This policy belongs in the kit because improving host detection later should
benefit every consumer without business-project changes.

## CI boundary

The reusable wails-desktop workflow intentionally targets the common
single-Wails-application case. It does not attempt to model arbitrary product
pre-build/post-build graphs.

Products with extra artifacts can keep custom orchestration while reusing the
Go/UI packages. Lower-level composite actions and the planned desktopkit CLI
can be added after migration experience shows the right extension points.
