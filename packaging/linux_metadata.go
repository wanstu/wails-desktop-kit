package packaging

import (
	"fmt"
	"os"
)

func linuxInstalledSizeKiB(request LinuxRequest) (int64, error) {
	files := []string{request.Input}
	for _, extra := range request.ExtraBinaries {
		files = append(files, extra.Source)
	}
	for _, path := range []string{request.DesktopFile, request.IconFile, request.CopyrightFile, request.AppStreamFile} {
		if path != "" {
			files = append(files, path)
		}
	}
	var total int64
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			return 0, fmt.Errorf("compute Debian installed size: %w", err)
		}
		total += info.Size()
	}
	if request.Systemd != nil {
		total += int64(len(linuxSystemdUnit(request)))
	}
	if total < 1024 {
		return 1, nil
	}
	return (total + 1023) / 1024, nil
}
