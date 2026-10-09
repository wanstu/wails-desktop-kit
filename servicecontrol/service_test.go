package servicecontrol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
)

type call struct {
	program string
	args    []string
}

func TestServiceControlsCommands(t *testing.T) {
	cases := []struct {
		action  string
		euid    int
		program string
		args    []string
	}{
		{"status", 1000, "systemctl", []string{"show", "--property=LoadState,ActiveState,UnitFileState", "--no-pager", "know-me-cli.service"}},
		{"start", 1000, "sudo", []string{"systemctl", "start", "know-me-cli.service"}},
		{"stop", 1000, "sudo", []string{"systemctl", "stop", "know-me-cli.service"}},
		{"restart", 1000, "sudo", []string{"systemctl", "restart", "know-me-cli.service"}},
		{"enable", 1000, "sudo", []string{"systemctl", "enable", "know-me-cli.service"}},
		{"disable", 1000, "sudo", []string{"systemctl", "disable", "know-me-cli.service"}},
		{"restart", 0, "systemctl", []string{"restart", "know-me-cli.service"}},
	}
	for _, tt := range cases {
		t.Run(fmt.Sprintf("%s-euid-%d", tt.action, tt.euid), func(t *testing.T) {
			var calls []call
			exec := func(_ context.Context, program string, args []string, out, errout io.Writer) error {
				calls = append(calls, call{program, append([]string{}, args...)})
				if tt.action == "status" {
					fmt.Fprint(out, "LoadState=loaded\nActiveState=active\nUnitFileState=enabled\n")
				}
				return nil
			}
			var output bytes.Buffer
			err := run(context.Background(), "linux", tt.euid, "know-me-cli.service", tt.action, &output, io.Discard, exec)
			if err != nil {
				t.Fatal(err)
			}
			want := []call{{tt.program, tt.args}}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("calls=%v want %v", calls, want)
			}
			if tt.action == "status" && !strings.Contains(output.String(), "Running: active") {
				t.Fatalf("status output: %q", output.String())
			}
		})
	}
}

func TestBadInputsNeverExecute(t *testing.T) {
	exec := func(context.Context, string, []string, io.Writer, io.Writer) error {
		t.Fatal("invalid arguments invoked executor")
		return nil
	}
	for _, tt := range []struct{ goos, unit, action string }{
		{"windows", "know-me-cli.service", "start"},
		{"linux", "../evil.service", "start"},
		{"linux", "sudo.service; id", "start"},
		{"linux", "know-me-cli.service", "delete"},
	} {
		if err := run(context.Background(), tt.goos, 1000, tt.unit, tt.action, io.Discard, io.Discard, exec); err == nil {
			t.Errorf("accepted %+v", tt)
		}
	}
}

func TestMissingOrInactiveService(t *testing.T) {
	for _, s := range []struct {
		payload   string
		wantError bool
	}{
		{"LoadState=not-found\nActiveState=inactive\nUnitFileState=\n", true},
		{"LoadState=loaded\nActiveState=inactive\nUnitFileState=disabled\n", false},
	} {
		var output bytes.Buffer
		err := run(context.Background(), "linux", 1000, "know-me-cli.service", "status", &output, io.Discard, func(_ context.Context, p string, a []string, out, errout io.Writer) error {
			_, e := fmt.Fprint(out, s.payload)
			return e
		})
		if (err != nil) != s.wantError {
			t.Errorf("result %v for %q", err, s.payload)
		}
		if !s.wantError && !strings.Contains(output.String(), "Running: inactive") {
			t.Errorf("unexpected inactive status: %s", output.String())
		}
	}
}

func TestSystemctlErrorsPropagate(t *testing.T) {
	denied := errors.New("permission denied")
	err := run(context.Background(), "linux", 1000, "know-me-cli.service", "restart", io.Discard, io.Discard, func(context.Context, string, []string, io.Writer, io.Writer) error { return denied })
	if !errors.Is(err, denied) {
		t.Fatalf("wrapped error: %v", err)
	}
}
