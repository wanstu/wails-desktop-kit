// Package servicecontrol provides a small, reusable Linux systemd CLI
// interface for applications installed by Kit's Debian service packaging.
//
// Service commands intentionally manage only the explicit, fixed unit name.
// They never evaluate a shell, accept arbitrary systemctl flags, or install
// and delete services.
package servicecontrol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var unitPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.@-]*\.service$`)

var supportedActions = map[string]bool{
	"status": true, "start": true, "stop": true, "restart": true,
	"enable": true, "disable": true,
}

// Actions is the list of CLI verbs supported for an installed service.
const Actions = "status|start|stop|restart|enable|disable"

type executor func(context.Context, string, []string, io.Writer, io.Writer) error

func osExecutor(ctx context.Context, program string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// Run manages an already installed Linux systemd unit. Mutating operations
// automatically request sudo on non-root Linux shells; status is read-only.
func Run(ctx context.Context, unit, action string, stdout, stderr io.Writer) error {
	return run(ctx, runtime.GOOS, os.Geteuid(), unit, action, stdout, stderr, osExecutor)
}

func run(ctx context.Context, goos string, euid int, unit, action string, stdout, stderr io.Writer, execute executor) error {
	if goos != "linux" {
		return errors.New("service management is available only on Linux (systemd)")
	}
	if !unitPattern.MatchString(unit) {
		return fmt.Errorf("invalid systemd service name %q", unit)
	}
	if !supportedActions[action] {
		return fmt.Errorf("unknown service command %q; expected %s", action, Actions)
	}
	if action == "status" {
		var output bytes.Buffer
		err := execute(ctx, "systemctl", []string{"show", "--property=LoadState,ActiveState,UnitFileState", "--no-pager", unit}, &output, stderr)
		if err != nil {
			return fmt.Errorf("unable to read %s status: %w", unit, err)
		}
		states := make(map[string]string)
		for _, line := range strings.Split(output.String(), "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
			if ok {
				states[key] = value
			}
		}
		if states["LoadState"] != "loaded" {
			return fmt.Errorf("service %s is not installed (load state: %s)", unit, states["LoadState"])
		}
		fmt.Fprintf(stdout, "Service: %s\nRunning: %s\nBoot:    %s\n", unit, states["ActiveState"], states["UnitFileState"])
		return nil
	}
	program, args := "systemctl", []string{action, unit}
	if euid != 0 {
		program = "sudo"
		args = append([]string{"systemctl"}, args...)
	}
	if err := execute(ctx, program, args, stdout, stderr); err != nil {
		return fmt.Errorf("service %s %s failed: %w", unit, action, err)
	}
	fmt.Fprintf(stdout, "%s: %s completed\n", unit, action)
	return nil
}
