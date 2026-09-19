package desktopkit

import "testing"

func TestParseLaunchOptionsWithAliases(t *testing.T) {
	got, err := ParseLaunchOptionsWithAliases([]string{"--minimized"}, "--minimized")
	if err != nil {
		t.Fatal(err)
	}
	if !got.AutoStart {
		t.Fatal("AutoStart = false")
	}
	if !HasAutoStartArg([]string{"app.exe", "--AUTOSTART"}) {
		t.Fatal("expected autostart arg")
	}
	if !HasAutoStartArg([]string{"--minimized"}, "minimized") {
		t.Fatal("expected alias")
	}
}

func TestParseLaunchOptionsRejectsUnknown(t *testing.T) {
	if _, err := ParseLaunchOptions([]string{"--unknown"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestShouldWakeSecondInstance(t *testing.T) {
	if shouldWakeSecondInstance(SecondInstanceIgnore, nil, nil) {
		t.Fatal("ignore policy woke window")
	}
	if shouldWakeSecondInstance(SecondInstanceWakeManual, []string{"--autostart"}, nil) {
		t.Fatal("autostart duplicate woke window")
	}
	if shouldWakeSecondInstance(SecondInstanceWakeManual, []string{"--minimized"}, []string{"--minimized"}) {
		t.Fatal("alias duplicate woke window")
	}
	if !shouldWakeSecondInstance(SecondInstanceWakeManual, []string{"file.txt"}, nil) {
		t.Fatal("manual duplicate did not wake window")
	}
}
