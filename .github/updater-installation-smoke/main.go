package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	if len(os.Args) == 3 && os.Args[1] == "--apply-update" {
		if err := applyUpdate(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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

func applyUpdate(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	launch, err := updater.InstallAndRestart("desktopkit-smoke", updater.DownloadResult{
		Asset: updater.Asset{
			Name: filepath.Base(path),
			Size: info.Size(),
		},
		Path:   path,
		SHA256: sum,
		Bytes:  info.Size(),
	})
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(launch)
}
