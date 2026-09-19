package main

import (
	"fmt"
	"strings"

	kitpackaging "github.com/wanstu/wails-desktop-kit/packaging"
)

type stringValues []string

func (v *stringValues) String() string { return strings.Join(*v, ",") }
func (v *stringValues) Set(value string) error {
	*v = append(*v, value)
	return nil
}

func parseExtraBinaries(values []string) ([]kitpackaging.LinuxBinary, error) {
	out := make([]kitpackaging.LinuxBinary, 0, len(values))
	for _, value := range values {
		name, path, ok := strings.Cut(value, "=")
		name = strings.TrimSpace(name)
		path = strings.TrimSpace(path)
		if !ok || name == "" || path == "" {
			return nil, fmt.Errorf("--extra-bin expects name=path, got %q", value)
		}
		out = append(out, kitpackaging.LinuxBinary{InstallName: name, Source: path})
	}
	return out, nil
}
