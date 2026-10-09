package packaging

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// LinuxSystemdService enables service packaging explicitly. Existing desktop
// consumers with nil Systemd retain the original packaging behavior.
type LinuxSystemdService struct {
	Name            string
	Description     string
	User            string
	Group           string
	DataDir         string
	Args            []string
	EnvironmentFile string   // Optional /etc/default/<service> path; absent files are ignored.
	Environment     []string // KEY=value entries; overridden by EnvironmentFile.
}

var linuxServiceUserPattern = regexp.MustCompile("^[a-z_][a-z0-9_-]*$")
var linuxServicePathPattern = regexp.MustCompile("^/[a-zA-Z0-9_+./-]+$")
var linuxServiceEnvKeyPattern = regexp.MustCompile("^[A-Za-z_][A-Za-z0-9_]*$")

func normalizedService(request LinuxRequest) LinuxSystemdService {
	s := *request.Systemd
	if s.Name == "" {
		s.Name = request.PackageName
	}
	if s.User == "" {
		s.User = request.PackageName
	}
	if s.Group == "" {
		s.Group = s.User
	}
	if s.DataDir == "" {
		s.DataDir = "/var/lib/" + request.PackageName
	}
	if s.Description == "" {
		s.Description = request.Description
	}
	return s
}
func validateLinuxSystemdService(request LinuxRequest) error {
	debSelected := false
	for _, format := range request.Formats {
		if format == FormatDeb {
			debSelected = true
		}
	}
	if !debSelected {
		return fmt.Errorf("systemd lifecycle requires deb in Linux packaging formats")
	}
	s := normalizedService(request)
	if !packageNamePattern.MatchString(s.Name) || strings.ContainsAny(s.Name, "+.") {
		return fmt.Errorf("invalid systemd service name %q", s.Name)
	}
	if !linuxServiceUserPattern.MatchString(s.User) || !linuxServiceUserPattern.MatchString(s.Group) {
		return fmt.Errorf("invalid service user or group: %q / %q", s.User, s.Group)
	}
	cleaned := path.Clean(s.DataDir)
	if !linuxServicePathPattern.MatchString(s.DataDir) || cleaned != s.DataDir ||
		s.DataDir == "/" || !strings.HasPrefix(s.DataDir, "/var/lib/") || s.DataDir == "/var/lib/" {
		return fmt.Errorf("invalid service data directory %q: require normalized absolute path under /var/lib", s.DataDir)
	}
	if strings.TrimSpace(s.Description) == "" || strings.ContainsAny(s.Description, "\r\n") {
		return fmt.Errorf("invalid service description")
	}
	if s.EnvironmentFile != "" {
		if !strings.HasPrefix(s.EnvironmentFile, "/etc/default/") || path.Clean(s.EnvironmentFile) != s.EnvironmentFile || !linuxServicePathPattern.MatchString(s.EnvironmentFile) {
			return fmt.Errorf("invalid service environment file %q", s.EnvironmentFile)
		}
	}
	for _, entry := range s.Environment {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 || !linuxServiceEnvKeyPattern.MatchString(parts[0]) || strings.ContainsAny(parts[1], "\r\n\x00") {
			return fmt.Errorf("invalid service environment variable")
		}
	}
	for _, arg := range s.Args {
		if arg == "" || strings.ContainsAny(arg, "\r\n\x00") {
			return fmt.Errorf("invalid systemd service argument %q", arg)
		}
	}
	return nil
}
func systemdExecArgument(arg string) string {
	arg = strings.ReplaceAll(arg, "%", "%%")
	arg = strings.ReplaceAll(arg, "\\", "\\\\")
	arg = strings.ReplaceAll(arg, "\"", "\\\"")
	return "\"" + arg + "\""
}
func linuxSystemdUnit(request LinuxRequest) []byte {
	s := normalizedService(request)
	args := append([]string{"/usr/bin/" + request.AppName}, s.Args...)
	for i := range args {
		args[i] = systemdExecArgument(args[i])
	}
	var environment strings.Builder
	for _, entry := range s.Environment {
		environment.WriteString("Environment=" + systemdExecArgument(entry) + "\n")
	}
	if s.EnvironmentFile != "" {
		environment.WriteString("EnvironmentFile=-" + s.EnvironmentFile + "\n")
	}
	body := fmt.Sprintf("[Unit]\nDescription=%s\nAfter=network-online.target\nWants=network-online.target\n\n[Service]\nType=simple\nUser=%s\nGroup=%s\nWorkingDirectory=%s\n%sExecStart=%s\nRestart=on-failure\nRestartSec=3\nUMask=0077\nNoNewPrivileges=true\n\n[Install]\nWantedBy=multi-user.target\n", s.Description, s.User, s.Group, s.DataDir, environment.String(), strings.Join(args, " "))
	return []byte(body)
}
func linuxServiceControlEntries(request LinuxRequest) []tarEntry {
	s := normalizedService(request)
	// Debian may pass the previously configured version even when reinstalling
	// a package after dpkg --remove. Record the removal explicitly so that the
	// next install starts the service, but regular upgrades still respect a
	// user's manually disabled service.
	removedMarker := path.Join(s.DataDir, ".desktopkit-systemd-removed")
	postinst := fmt.Sprintf(`#!/bin/sh
set -e
if [ "$1" = "configure" ]; then
  if ! getent group '%s' >/dev/null 2>&1; then
    groupadd --system '%s'
  fi
  if ! getent passwd '%s' >/dev/null 2>&1; then
    useradd --system --gid '%s' --home-dir '%s' --shell /usr/sbin/nologin '%s'
  fi
  install -d -m 0700 -o '%s' -g '%s' '%s'
  if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    systemctl daemon-reload
    if [ -z "${2:-}" ] || [ -f '%s' ]; then
      systemctl enable '%s.service'
      systemctl start '%s.service'
      rm -f -- '%s'
    else
      systemctl try-restart '%s.service'
    fi
  fi
fi
exit 0
`, s.Group, s.Group, s.User, s.Group, s.DataDir, s.User,
		s.User, s.Group, s.DataDir, removedMarker, s.Name, s.Name, removedMarker, s.Name)
	prerm := fmt.Sprintf(`#!/bin/sh
set -e
if [ "$1" = "remove" ]; then
  if [ -d '%s' ]; then
    touch '%s'
  fi
  if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
    systemctl disable --now '%s.service'
  fi
fi
exit 0
`, s.DataDir, removedMarker, s.Name)
	postrm := fmt.Sprintf(`#!/bin/sh
set -e
if [ "$1" = "purge" ]; then
  rm -f -- '%s'
fi
if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
  systemctl daemon-reload
fi
# Preserve all application data, encryption keys and databases on remove/purge.
exit 0
`, removedMarker)
	return []tarEntry{
		{name: "./postinst", data: []byte(postinst), mode: 0755},
		{name: "./prerm", data: []byte(prerm), mode: 0755},
		{name: "./postrm", data: []byte(postrm), mode: 0755},
	}
}
func linuxServiceDataEntry(request LinuxRequest) tarEntry {
	s := normalizedService(request)
	return tarEntry{name: "./lib/systemd/system/" + s.Name + ".service", data: linuxSystemdUnit(request), mode: 0644}
}
