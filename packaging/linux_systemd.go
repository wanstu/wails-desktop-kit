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
	Name        string
	Description string
	User        string
	Group       string
	DataDir     string
	Args        []string
}

var linuxServiceUserPattern = regexp.MustCompile("^[a-z_][a-z0-9_-]*$")
var linuxServicePathPattern = regexp.MustCompile("^/[a-zA-Z0-9_+./-]+$")

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
	body := fmt.Sprintf("[Unit]\nDescription=%s\nAfter=network-online.target\nWants=network-online.target\n\n[Service]\nType=simple\nUser=%s\nGroup=%s\nWorkingDirectory=%s\nExecStart=%s\nRestart=on-failure\nRestartSec=3\nUMask=0077\nNoNewPrivileges=true\n\n[Install]\nWantedBy=multi-user.target\n", s.Description, s.User, s.Group, s.DataDir, strings.Join(args, " "))
	return []byte(body)
}
func linuxServiceControlEntries(request LinuxRequest) []tarEntry {
	s := normalizedService(request)
	postinst := fmt.Sprintf("#!/bin/sh\nset -e\nif [ \"$1\" = \"configure\" ]; then\n  if ! getent group '%s' >/dev/null 2>&1; then\n    groupadd --system '%s'\n  fi\n  if ! getent passwd '%s' >/dev/null 2>&1; then\n    useradd --system --gid '%s' --home-dir '%s' --shell /usr/sbin/nologin '%s'\n  fi\n  install -d -m 0700 -o '%s' -g '%s' '%s'\n  if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then\n    systemctl daemon-reload\n    if [ -z \"${2:-}\" ]; then\n      systemctl enable '%s.service'\n      systemctl start '%s.service'\n    else\n      systemctl try-restart '%s.service'\n    fi\n  fi\nfi\nexit 0\n", s.Group, s.Group, s.User, s.Group, s.DataDir, s.User, s.User, s.Group, s.DataDir, s.Name, s.Name, s.Name)
	prerm := fmt.Sprintf("#!/bin/sh\nset -e\nif [ \"$1\" = \"remove\" ] && command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then\n  systemctl disable --now '%s.service'\nfi\nexit 0\n", s.Name)
	return []tarEntry{
		{name: "./postinst", data: []byte(postinst), mode: 0755},
		{name: "./prerm", data: []byte(prerm), mode: 0755},
		{name: "./postrm", data: []byte("#!/bin/sh\nset -e\nif command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then\n  systemctl daemon-reload\nfi\n# Preserve all application data, encryption keys and databases on remove/purge.\nexit 0\n"), mode: 0755},
	}
}
func linuxServiceDataEntry(request LinuxRequest) tarEntry {
	s := normalizedService(request)
	return tarEntry{name: "./lib/systemd/system/" + s.Name + ".service", data: linuxSystemdUnit(request), mode: 0644}
}
