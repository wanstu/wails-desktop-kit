//go:build windows

package autostart

import "testing"

func TestWindowsCommandQuotesExecutableAndArguments(t *testing.T) {
	manager, err := New(Config{
		ID:             "example",
		DisplayName:    "Example",
		ExecutablePath: `C:\Program Files\Example\example.exe`,
		Arguments:      []string{"--autostart", "profile one"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := manager.command()
	if err != nil {
		t.Fatal(err)
	}
	want := `"C:\Program Files\Example\example.exe" --autostart "profile one"`
	if got != want {
		t.Fatalf("command = %q, want %q", got, want)
	}
}
