# FRP Client Manager migration target

FRP Client Manager is the first intended consumer because it already has
Windows/Linux/macOS CI and a Debian 13 runtime validation baseline.

The migration must preserve behavior.

## Replace

After migration, FRP should no longer own:

- tray_manager_supported.go and tray_manager_other.go;
- internal/autostart platform implementations;
- common Wails shell plumbing in cmd/frp-client-desktop/main.go;
- duplicate standard CI matrix/release staging logic;
- generic CSS primitives that directly match kit components.

## Keep

FRP-specific behavior remains in FRP:

- profile management;
- frpc process lifecycle;
- FRP config parser/editor;
- start/stop/restart-all tray actions;
- quit-and-stop-frpc behavior;
- FRP settings persistence.

## Acceptance

Migration is complete only when:

- Windows tests/build pass.
- Linux tests/build pass.
- macOS universal build passes.
- Debian 13 launch/tray/autostart remains usable.
- Multiple FRP profiles still run independently.
- Normal exit preserves frpc.
- Explicit stop-and-exit stops managed frpc processes.
