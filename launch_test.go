package desktopkit

import "testing"

func TestParseLaunchOptions(t *testing.T) {
	got, err := ParseLaunchOptions([]string{"--autostart"})
	if err != nil {
		t.Fatal(err)
	}
	if !got.AutoStart {
		t.Fatal("AutoStart = false, want true")
	}
}

func TestParseLaunchOptionsRejectsUnexpectedArgs(t *testing.T) {
	if _, err := ParseLaunchOptions([]string{"unexpected"}); err == nil {
		t.Fatal("expected error")
	}
}
