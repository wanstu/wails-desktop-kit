package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wanstu/wails-desktop-kit/updater"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--hold" {
		time.Sleep(60 * time.Second)
		return
	}

	state, err := updater.CurrentInstallation("desktopkit-smoke")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if state.Managed() {
		marker := filepath.Join(filepath.Dir(state.CurrentExecutable), "restart-smoke.txt")
		if err := os.WriteFile(marker, []byte("started"), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(state); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
