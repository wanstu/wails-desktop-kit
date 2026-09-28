package updater

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type InstallationMode string

const (
	InstallationPortable InstallationMode = "portable"
	InstallationUser     InstallationMode = "user"
	InstallationMachine  InstallationMode = "machine"
)

type Installation struct {
	AppID             string
	Mode              InstallationMode
	CurrentExecutable string
	InstallDir        string
	Uninstaller       string
	DisplayVersion    string
}

func (installation Installation) Managed() bool {
	return installation.Mode == InstallationUser || installation.Mode == InstallationMachine
}

func CurrentInstallation(appID string) (Installation, error) {
	appID = strings.TrimSpace(appID)
	if appID == "" {
		return Installation{}, errors.New("app id is required")
	}
	if strings.ContainsAny(appID, `/\`) {
		return Installation{}, errors.New("app id must not contain path separators")
	}

	executable, err := os.Executable()
	if err != nil {
		return Installation{}, err
	}
	executable, err = filepath.Abs(executable)
	if err != nil {
		return Installation{}, err
	}
	executable = filepath.Clean(executable)

	return currentInstallation(appID, executable)
}
