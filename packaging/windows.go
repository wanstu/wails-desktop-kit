package packaging

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

const FormatWindowsSetup Format = "setup"

var windowsVersionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)`)

type WindowsRequest struct {
	Input             string
	OutputDir         string
	AppName           string
	AssetBase         string
	PackageVersion    string
	Architecture      string
	InstallScope      string
	ProductName       string
	Publisher         string
	AppID             string
	IconFile          string
	StartMenuShortcut bool
	DesktopShortcut   bool
	NSISPath          string
}

type windowsTemplateData struct {
	Input             string
	Output            string
	AppExe            string
	AppID             string
	ProductName       string
	Publisher         string
	DisplayVersion    string
	NumericVersion    string
	InstallScope      string
	IconFile          string
	StartMenuShortcut bool
	DesktopShortcut   bool
}

var compileWindowsInstaller = func(executable, scriptPath string) error {
	cmd := exec.Command(executable, "/V2", scriptPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("run NSIS compiler: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func PackageWindows(request WindowsRequest) (Artifact, error) {
	request = normalizeWindowsRequest(request)
	if err := validateWindowsRequest(request); err != nil {
		return Artifact{}, err
	}
	if err := os.MkdirAll(request.OutputDir, 0o755); err != nil {
		return Artifact{}, fmt.Errorf("create output dir: %w", err)
	}

	input, err := filepath.Abs(request.Input)
	if err != nil {
		return Artifact{}, fmt.Errorf("resolve input: %w", err)
	}
	output := filepath.Join(request.OutputDir, request.AssetBase+"-windows-"+request.Architecture+"-setup.exe")
	output, err = filepath.Abs(output)
	if err != nil {
		return Artifact{}, fmt.Errorf("resolve output: %w", err)
	}
	icon := ""
	if request.IconFile != "" {
		icon, err = filepath.Abs(request.IconFile)
		if err != nil {
			return Artifact{}, fmt.Errorf("resolve icon: %w", err)
		}
	}

	nsis := request.NSISPath
	if nsis == "" {
		nsis, err = exec.LookPath("makensis")
		if err != nil {
			return Artifact{}, errors.New("makensis was not found; install NSIS or pass --nsis")
		}
	}

	tempDir, err := os.MkdirTemp("", "desktopkit-nsis-*")
	if err != nil {
		return Artifact{}, fmt.Errorf("create NSIS temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	scriptPath := filepath.Join(tempDir, "installer.nsi")
	data := windowsTemplateData{
		Input:             nsisEscape(input),
		Output:            nsisEscape(output),
		AppExe:            nsisEscape(request.AppName + ".exe"),
		AppID:             nsisEscape(request.AppID),
		ProductName:       nsisEscape(request.ProductName),
		Publisher:         nsisEscape(request.Publisher),
		DisplayVersion:    nsisEscape(request.PackageVersion),
		NumericVersion:    numericWindowsVersion(request.PackageVersion),
		InstallScope:      request.InstallScope,
		IconFile:          nsisEscape(icon),
		StartMenuShortcut: request.StartMenuShortcut,
		DesktopShortcut:   request.DesktopShortcut,
	}
	if err := writeWindowsInstallerScript(scriptPath, data); err != nil {
		return Artifact{}, err
	}
	if err := compileWindowsInstaller(nsis, scriptPath); err != nil {
		return Artifact{}, err
	}
	if info, err := os.Stat(output); err != nil {
		return Artifact{}, fmt.Errorf("installer was not created at %s: %w", output, err)
	} else if !info.Mode().IsRegular() || info.Size() == 0 {
		return Artifact{}, fmt.Errorf("installer output is empty: %s", output)
	}
	return checksumArtifact(output, FormatWindowsSetup)
}

func normalizeWindowsRequest(request WindowsRequest) WindowsRequest {
	request.Input = filepath.Clean(strings.TrimSpace(request.Input))
	request.OutputDir = filepath.Clean(strings.TrimSpace(request.OutputDir))
	request.AppName = strings.TrimSpace(request.AppName)
	request.AssetBase = strings.TrimSpace(request.AssetBase)
	request.PackageVersion = strings.TrimSpace(strings.TrimPrefix(request.PackageVersion, "v"))
	request.Architecture = strings.ToLower(strings.TrimSpace(request.Architecture))
	request.InstallScope = strings.ToLower(strings.TrimSpace(request.InstallScope))
	request.ProductName = strings.TrimSpace(request.ProductName)
	request.Publisher = strings.TrimSpace(request.Publisher)
	request.AppID = strings.TrimSpace(request.AppID)
	request.IconFile = strings.TrimSpace(request.IconFile)
	request.NSISPath = strings.TrimSpace(request.NSISPath)
	if request.OutputDir == "" || request.OutputDir == "." {
		request.OutputDir = "dist"
	}
	if request.AssetBase == "" {
		request.AssetBase = request.AppName
	}
	if request.PackageVersion == "" {
		request.PackageVersion = "0.0.0"
	}
	if request.Architecture == "" {
		request.Architecture = "amd64"
	}
	if request.InstallScope == "" {
		request.InstallScope = "user"
	}
	if request.ProductName == "" {
		request.ProductName = request.AppName
	}
	if request.Publisher == "" {
		request.Publisher = "Unknown"
	}
	if request.AppID == "" {
		request.AppID = request.AppName
	}
	return request
}

func validateWindowsRequest(request WindowsRequest) error {
	if request.Input == "" || request.Input == "." {
		return errors.New("input executable is required")
	}
	info, err := os.Stat(request.Input)
	if err != nil {
		return fmt.Errorf("stat input executable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("input must be a regular file")
	}
	if !appNamePattern.MatchString(request.AppName) {
		return fmt.Errorf("invalid app name %q", request.AppName)
	}
	if request.AssetBase == "" || strings.ContainsAny(request.AssetBase, `/\`) {
		return fmt.Errorf("invalid asset base %q", request.AssetBase)
	}
	if request.Architecture != "amd64" && request.Architecture != "arm64" {
		return fmt.Errorf("unsupported architecture %q", request.Architecture)
	}
	if request.InstallScope != "user" && request.InstallScope != "machine" {
		return fmt.Errorf("unsupported install scope %q (supported: user, machine)", request.InstallScope)
	}
	for name, value := range map[string]string{
		"package version": request.PackageVersion,
		"product name":    request.ProductName,
		"publisher":       request.Publisher,
		"app id":          request.AppID,
	} {
		if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("invalid %s", name)
		}
	}
	if strings.ContainsAny(request.AppID, `/\`) {
		return fmt.Errorf("invalid app id %q", request.AppID)
	}
	if request.IconFile != "" {
		info, err := os.Stat(request.IconFile)
		if err != nil {
			return fmt.Errorf("stat icon file: %w", err)
		}
		if !info.Mode().IsRegular() {
			return errors.New("icon file must be a regular file")
		}
		if !strings.EqualFold(filepath.Ext(request.IconFile), ".ico") {
			return errors.New("Windows installer icon must be an .ico file")
		}
	}
	return nil
}

func numericWindowsVersion(version string) string {
	match := windowsVersionPattern.FindStringSubmatch(strings.TrimSpace(version))
	if len(match) != 4 {
		return "0.0.0.0"
	}
	return match[1] + "." + match[2] + "." + match[3] + ".0"
}

func nsisEscape(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "$", "$$")
	value = strings.ReplaceAll(value, `"`, `$\"`)
	return value
}

func writeWindowsInstallerScript(path string, data windowsTemplateData) error {
	var output bytes.Buffer
	if err := windowsInstallerTemplate.Execute(&output, data); err != nil {
		return fmt.Errorf("render NSIS script: %w", err)
	}
	if err := os.WriteFile(path, output.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write NSIS script: %w", err)
	}
	return nil
}

var windowsInstallerTemplate = template.Must(template.New("windows-installer").Parse(`Unicode true
SetCompressor /SOLID lzma
!include "MUI2.nsh"

!define APP_EXE "{{.AppExe}}"
!define APP_ID "{{.AppID}}"
!define PRODUCT_NAME "{{.ProductName}}"
!define PUBLISHER "{{.Publisher}}"
!define DISPLAY_VERSION "{{.DisplayVersion}}"

Name "${PRODUCT_NAME}"
OutFile "{{.Output}}"
VIProductVersion "{{.NumericVersion}}"
VIAddVersionKey /LANG=1033 "ProductName" "${PRODUCT_NAME}"
VIAddVersionKey /LANG=1033 "ProductVersion" "${DISPLAY_VERSION}"
VIAddVersionKey /LANG=1033 "CompanyName" "${PUBLISHER}"
VIAddVersionKey /LANG=1033 "FileDescription" "${PRODUCT_NAME} Installer"
{{if .IconFile}}Icon "{{.IconFile}}"{{end}}

{{if eq .InstallScope "user"}}RequestExecutionLevel user
InstallDir "$LOCALAPPDATA\Programs\${PRODUCT_NAME}"
{{else}}RequestExecutionLevel admin
InstallDir "$PROGRAMFILES64\${PUBLISHER}\${PRODUCT_NAME}"
{{end}}

!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOverwrite on
  {{if eq .InstallScope "user"}}SetShellVarContext current{{else}}SetShellVarContext all
  SetRegView 64{{end}}
  SetOutPath "$INSTDIR"
  File /oname=${APP_EXE} "{{.Input}}"
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  {{if .StartMenuShortcut}}CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\${APP_EXE}"
  CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk" "$INSTDIR\Uninstall.exe"{{end}}
  {{if .DesktopShortcut}}CreateShortcut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\${APP_EXE}"{{end}}

  {{if eq .InstallScope "user"}}WriteRegStr HKCU{{else}}WriteRegStr HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayName" "${PRODUCT_NAME}"
  {{if eq .InstallScope "user"}}WriteRegStr HKCU{{else}}WriteRegStr HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayVersion" "${DISPLAY_VERSION}"
  {{if eq .InstallScope "user"}}WriteRegStr HKCU{{else}}WriteRegStr HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "Publisher" "${PUBLISHER}"
  {{if eq .InstallScope "user"}}WriteRegStr HKCU{{else}}WriteRegStr HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "InstallLocation" "$INSTDIR"
  {{if eq .InstallScope "user"}}WriteRegStr HKCU{{else}}WriteRegStr HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "DisplayIcon" "$INSTDIR\${APP_EXE}"
  {{if eq .InstallScope "user"}}WriteRegStr HKCU{{else}}WriteRegStr HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  {{if eq .InstallScope "user"}}WriteRegStr HKCU{{else}}WriteRegStr HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "QuietUninstallString" '"$INSTDIR\Uninstall.exe" /S'
  {{if eq .InstallScope "user"}}WriteRegDWORD HKCU{{else}}WriteRegDWORD HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NoModify" 1
  {{if eq .InstallScope "user"}}WriteRegDWORD HKCU{{else}}WriteRegDWORD HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}" "NoRepair" 1
SectionEnd

Section "Uninstall"
  {{if eq .InstallScope "user"}}SetShellVarContext current{{else}}SetShellVarContext all
  SetRegView 64{{end}}
  {{if .StartMenuShortcut}}Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall.lnk"
  RMDir "$SMPROGRAMS\${PRODUCT_NAME}"{{end}}
  {{if .DesktopShortcut}}Delete "$DESKTOP\${PRODUCT_NAME}.lnk"{{end}}
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  {{if eq .InstallScope "user"}}DeleteRegKey HKCU{{else}}DeleteRegKey HKLM{{end}} "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_ID}"
SectionEnd
`))
