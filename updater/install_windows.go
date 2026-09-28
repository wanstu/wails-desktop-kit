//go:build windows

package updater

import (
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func automaticInstallSupported() bool {
	return true
}

func launchUpdateSetup(setupPath string, currentPID int) (int, error) {
	command := exec.Command(
		setupPath,
		"/S",
		"/WAITPID="+strconv.Itoa(currentPID),
		"/RESTART=1",
	)
	command.Dir = filepath.Dir(setupPath)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		return 0, err
	}
	pid := command.Process.Pid
	_ = command.Process.Release()
	return pid, nil
}
