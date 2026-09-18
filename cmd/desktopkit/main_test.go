package main

import "testing"

func TestRunHelp(t *testing.T) {
	if err := run([]string{"--help"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	if err := run([]string{"nope"}); err == nil {
		t.Fatal("expected unknown command error")
	}
}
