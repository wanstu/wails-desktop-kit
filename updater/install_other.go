//go:build !windows

package updater

func automaticInstallSupported() bool {
	return false
}

func launchUpdateSetup(string, int) (int, error) {
	return 0, ErrInstallUnsupported
}
