//go:build !windows

package updater

func currentInstallation(appID, currentExecutable string) (Installation, error) {
	return Installation{
		AppID:             appID,
		Mode:              InstallationPortable,
		CurrentExecutable: currentExecutable,
	}, nil
}
