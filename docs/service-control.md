# Linux CLI service controls

Starting in Kit v0.11.1, applications that package an installed Linux systemd service can expose friendly CLI commands with the reusable `servicecontrol` package.

```go
import "github.com/wanstu/wails-desktop-kit/servicecontrol"

err := servicecontrol.Run(ctx, "my-app.service", action, os.Stdout, os.Stderr)
```

`action` is one of `status`, `start`, `stop`, `restart`, `enable` and `disable`.

The helper checks the operating system and service unit name; it only runs fixed systemctl verbs and never evaluates a shell. `status` shows running and startup states even when a service is stopped or disabled, but returns an error when the unit does not exist. Mutating commands use `sudo` on non-root Linux shells and may request a password.

**Packaging and runtime control are separate.** `desktopkit package linux --systemd` generates a Debian package with service lifecycle scripts; the application must explicitly wire the user-facing CLI verb into `servicecontrol.Run`. Installing an older application binary does not magically add the command.

To actually verify a product supports these commands, run the *installed executable* (not only library unit tests) on a systemd-based Linux system and test both `service status` and a mutating action.
