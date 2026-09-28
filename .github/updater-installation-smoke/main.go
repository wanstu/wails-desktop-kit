package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/wanstu/wails-desktop-kit/updater"
)

func main() {
	state, err := updater.CurrentInstallation("desktopkit-smoke")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := json.NewEncoder(os.Stdout).Encode(state); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
