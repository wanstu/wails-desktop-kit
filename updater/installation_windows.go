//go:build windows

package updater

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const windowsUninstallBase = `Software\Microsoft\Windows\CurrentVersion\Uninstall`

type windowsInstallRecord struct {
	mode           InstallationMode
	installDir     string
	displayIcon    string
	displayVersion string
	uninstaller    string
}

type windowsRegistryView struct {
	root   registry.Key
	mode   InstallationMode
	access uint32
}

func currentInstallation(appID, currentExecutable string) (Installation, error) {
	portable := Installation{
		AppID:             appID,
		Mode:              InstallationPortable,
		CurrentExecutable: currentExecutable,
	}

	records, err := readWindowsInstallRecords(appID)
	if err != nil {
		return Installation{}, err
	}
	for _, record := range records {
		if !sameWindowsPath(filepath.Dir(currentExecutable), record.installDir) {
			continue
		}
		if !sameWindowsPath(currentExecutable, record.displayIcon) {
			continue
		}
		info, err := os.Stat(record.uninstaller)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		return Installation{
			AppID:             appID,
			Mode:              record.mode,
			CurrentExecutable: currentExecutable,
			InstallDir:        filepath.Clean(record.installDir),
			Uninstaller:       filepath.Clean(record.uninstaller),
			DisplayVersion:    strings.TrimSpace(record.displayVersion),
		}, nil
	}
	return portable, nil
}

func readWindowsInstallRecords(appID string) ([]windowsInstallRecord, error) {
	path := windowsUninstallBase + `\` + appID
	views := []windowsRegistryView{
		{root: registry.CURRENT_USER, mode: InstallationUser, access: registry.QUERY_VALUE | registry.WOW64_64KEY},
		{root: registry.CURRENT_USER, mode: InstallationUser, access: registry.QUERY_VALUE | registry.WOW64_32KEY},
		{root: registry.LOCAL_MACHINE, mode: InstallationMachine, access: registry.QUERY_VALUE | registry.WOW64_64KEY},
		{root: registry.LOCAL_MACHINE, mode: InstallationMachine, access: registry.QUERY_VALUE | registry.WOW64_32KEY},
	}

	records := make([]windowsInstallRecord, 0, len(views))
	seen := map[string]bool{}
	for _, view := range views {
		key, err := registry.OpenKey(view.root, path, view.access)
		if errors.Is(err, registry.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("open Windows uninstall registry: %w", err)
		}

		installDir, _, installErr := key.GetStringValue("InstallLocation")
		displayIcon, _, iconErr := key.GetStringValue("DisplayIcon")
		displayVersion, _, _ := key.GetStringValue("DisplayVersion")
		_ = key.Close()

		if installErr != nil || iconErr != nil {
			continue
		}
		installDir = filepath.Clean(strings.TrimSpace(installDir))
		displayIcon = cleanWindowsRegistryPath(displayIcon)
		if installDir == "." || installDir == "" || displayIcon == "" {
			continue
		}
		uninstaller := filepath.Join(installDir, "Uninstall.exe")
		id := string(view.mode) + "|" + strings.ToLower(installDir) + "|" + strings.ToLower(displayIcon)
		if seen[id] {
			continue
		}
		seen[id] = true
		records = append(records, windowsInstallRecord{
			mode:           view.mode,
			installDir:     installDir,
			displayIcon:    displayIcon,
			displayVersion: displayVersion,
			uninstaller:    uninstaller,
		})
	}
	return records, nil
}

func cleanWindowsRegistryPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	if index := strings.LastIndex(value, ","); index >= 0 {
		suffix := strings.TrimSpace(value[index+1:])
		allDigits := suffix != ""
		for _, r := range suffix {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			value = strings.TrimSpace(value[:index])
			value = strings.Trim(value, `"`)
		}
	}
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}

func sameWindowsPath(left, right string) bool {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)
	if left == "" || right == "" {
		return false
	}
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil {
		left = leftAbs
	}
	if rightErr == nil {
		right = rightAbs
	}
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}
