//go:build linux

package autostart

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestAutoStartArgumentHelper(t *testing.T) {
	output := os.Getenv("DESKTOPKIT_AUTOSTART_ARG_OUTPUT")
	if output == "" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			data, err := json.Marshal(os.Args[i+1:])
			if err != nil {
				os.Exit(2)
			}
			if os.WriteFile(output, data, 0600) != nil {
				os.Exit(3)
			}
			os.Exit(0)
		}
	}
	os.Exit(4)
}
func TestDesktopEntryLaunchPreservesArguments(t *testing.T) {
	gio, err := exec.LookPath("gio")
	if err != nil {
		t.Skip("gio is required for desktop entry integration")
	}
	dir := t.TempDir()
	output := filepath.Join(dir, "argv.json")
	t.Setenv("DESKTOPKIT_AUTOSTART_ARG_OUTPUT", output)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"space a", "quote\"x", "back\\slash", "$HOME", "%f", "back" + string(rune(96)) + "tick", "中文", "tab\tX", ""}
	args := append([]string{"-test.run=TestAutoStartArgumentHelper", "--"}, want...)
	manager, err := New(Config{ID: "example", ExecutablePath: exe, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	content, err := manager.linuxContent()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "example.desktop")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	if data, err := exec.Command(gio, "launch", path).CombinedOutput(); err != nil {
		t.Fatalf("gio launch: %v %s", err, data)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(output)
		if err == nil {
			var got []string
			if json.Unmarshal(data, &got) == nil {
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("argv=%q want=%q", got, want)
				}
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("desktop launcher did not produce argument output")
}
func TestRejectsMultilineExec(t *testing.T) {
	manager, _ := New(Config{ID: "example", ExecutablePath: "/opt/app", Arguments: []string{"bad\narg"}})
	if _, err := manager.linuxContent(); err == nil {
		t.Fatal("accepted multiline argument")
	}
	t.Setenv("XDG_CONFIG_HOME", "relative")
	if _, err := manager.linuxPath(); err == nil {
		t.Fatal("accepted relative config home")
	}
}
