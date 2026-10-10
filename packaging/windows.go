package packaging

import (
	"bytes"
	_ "embed"
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
	Input              string
	OutputDir          string
	AppName            string
	AssetBase          string
	PackageVersion     string
	Architecture       string
	InstallScope       string
	ProductName        string
	Publisher          string
	AppID              string
	IconFile           string
	StartMenuShortcut  bool
	DesktopShortcut    bool
	NSISPath           string
	ConfirmStopRunning bool // Opt-in: installer asks before stopping the installed app's processes.
}

type windowsTemplateData struct {
	Input              string
	Output             string
	AppExe             string
	AppID              string
	ProductName        string
	Publisher          string
	DisplayVersion     string
	NumericVersion     string
	InstallScope       string
	IconFile           string
	StartMenuShortcut  bool
	DesktopShortcut    bool
	ConfirmStopRunning bool
	ShutdownScript     string
}

//go:embed windows_installer_stop.ps1
var windowsInstallerStopScript []byte

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
	shutdownScript := ""
	if request.ConfirmStopRunning {
		shutdownScript = filepath.Join(tempDir, "kit-stop.ps1")
		if err := os.WriteFile(shutdownScript, windowsInstallerStopScript, 0o600); err != nil {
			return Artifact{}, fmt.Errorf("write installer shutdown script: %w", err)
		}
	}
	data := windowsTemplateData{
		Input:              nsisEscape(input),
		Output:             nsisEscape(output),
		AppExe:             nsisEscape(request.AppName + ".exe"),
		AppID:              nsisEscape(request.AppID),
		ProductName:        nsisEscape(request.ProductName),
		Publisher:          nsisEscape(request.Publisher),
		DisplayVersion:     nsisEscape(request.PackageVersion),
		NumericVersion:     numericWindowsVersion(request.PackageVersion),
		InstallScope:       request.InstallScope,
		IconFile:           nsisEscape(icon),
		StartMenuShortcut:  request.StartMenuShortcut,
		DesktopShortcut:    request.DesktopShortcut,
		ConfirmStopRunning: request.ConfirmStopRunning,
		ShutdownScript:     nsisEscape(shutdownScript),
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
	// makensis treats scripts without a BOM as the host ANSI codepage even
	// when Unicode true is set, causing Chinese installer text to become mojibake.
	// Mark the generated .nsi as UTF-8 explicitly.
	utf8Script := append([]byte{0xEF, 0xBB, 0xBF}, output.Bytes()...)
	if err := os.WriteFile(path, utf8Script, 0o644); err != nil {
		return fmt.Errorf("write NSIS script: %w", err)
	}
	return nil
}

var windowsInstallerTemplate = template.Must(template.New("windows-installer").Parse(`Unicode true
SetCompressor /SOLID lzma
!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"

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
VIAddVersionKey /LANG=2052 "FileDescription" "${PRODUCT_NAME} 安装程序"
{{if .IconFile}}Icon "{{.IconFile}}"{{end}}

{{if eq .InstallScope "user"}}RequestExecutionLevel user
InstallDir "$LOCALAPPDATA\Programs\${PRODUCT_NAME}"
{{else}}RequestExecutionLevel admin
InstallDir "$PROGRAMFILES64\${PUBLISHER}\${PRODUCT_NAME}"
{{end}}

!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "$(DesktopKitWelcomeTitle)"
!define MUI_WELCOMEPAGE_TEXT "$(DesktopKitWelcomeText)"
!define MUI_FINISHPAGE_TITLE "$(DesktopKitFinishTitle)"
!define MUI_FINISHPAGE_TEXT "$(DesktopKitFinishText)"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_UNFINISHPAGE_NOAUTOCLOSE
{{if eq .InstallScope "user"}}!define MUI_FINISHPAGE_RUN "$INSTDIR\${APP_EXE}"
{{end}}!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
!insertmacro MUI_LANGUAGE "SimpChinese"

LangString DesktopKitWelcomeTitle ${LANG_ENGLISH} "Welcome to ${PRODUCT_NAME}"
LangString DesktopKitWelcomeTitle ${LANG_SIMPCHINESE} "欢迎安装 ${PRODUCT_NAME}"
LangString DesktopKitWelcomeText ${LANG_ENGLISH} "This wizard installs ${PRODUCT_NAME} on your computer.$\r$\n$\r$\nYour personal data will be kept separate from the app and preserved when upgrading."
LangString DesktopKitWelcomeText ${LANG_SIMPCHINESE} "此向导将安装 ${PRODUCT_NAME}。$\r$\n$\r$\n用户数据与程序分开保存，覆盖升级不会清除用户数据。"
LangString DesktopKitFinishTitle ${LANG_ENGLISH} "${PRODUCT_NAME} installation complete"
LangString DesktopKitFinishTitle ${LANG_SIMPCHINESE} "${PRODUCT_NAME} 已安装完成"
LangString DesktopKitFinishText ${LANG_ENGLISH} "${PRODUCT_NAME} is ready. Click Finish to close Setup."
LangString DesktopKitFinishText ${LANG_SIMPCHINESE} "${PRODUCT_NAME} 已准备就绪。点击“完成”退出安装程序。"
LangString DesktopKitCloseApp ${LANG_ENGLISH} "${PRODUCT_NAME} is still running. Please exit the app (including its tray icon) and click Retry. Cancel will leave the existing installation unchanged."
LangString DesktopKitCloseApp ${LANG_SIMPCHINESE} "${PRODUCT_NAME} 仍在运行。请先从托盘菜单彻底退出程序，然后点击“重试”。点击“取消”将保留现有安装。"
{{if .ConfirmStopRunning}}LangString DesktopKitConfirmStop ${LANG_ENGLISH} "${PRODUCT_NAME} is running. Close the installed application and its background processes to continue? Unsaved work and running tasks may be lost. Nothing is closed without your confirmation."
LangString DesktopKitConfirmStop ${LANG_SIMPCHINESE} "${PRODUCT_NAME} 仍在运行。是否关闭正在运行的程序及其后台进程，然后继续安装？这可能中断正在执行的任务，请先保存工作。未经确认不会关闭任何进程。"
LangString DesktopKitStopFailed ${LANG_ENGLISH} "Unable to close the installed application safely. Setup will stop without modifying the installation. Please close the application and retry."
LangString DesktopKitStopFailed ${LANG_SIMPCHINESE} "未能关闭已安装的程序。安装未开始，现有安装保持不变。请检查正在运行的程序后重试。"
{{end}}

Var DesktopKitRestartAfterInstall

Function .onInit
  ${GetParameters} $0

  ${GetOptions} $0 "/WAITPID=" $1
  StrCmp $1 "" desktopkit_wait_done
  System::Call 'kernel32::OpenProcess(i 0x00100000, i 0, i r1) i .r2'
  StrCmp $2 0 desktopkit_wait_done
  System::Call 'kernel32::WaitForSingleObject(i r2, i 120000) i .r3'
  System::Call 'kernel32::CloseHandle(i r2)'
  StrCmp $3 0 desktopkit_wait_done
  SetErrorLevel 2
  Quit

desktopkit_wait_done:
  ${GetOptions} $0 "/RESTART=" $1
  StrCmp $1 "1" 0 desktopkit_init_done
  StrCpy $DesktopKitRestartAfterInstall "1"

desktopkit_init_done:
FunctionEnd

Section "Install"
  ; Check before writing any files, including when /S is used.
  ; This avoids the default obscure Abort/Retry/Ignore locked-EXE dialog.
  IfFileExists "$INSTDIR\${APP_EXE}" 0 desktopkit_install_process_ok
 desktopkit_check_process:
  nsExec::ExecToStack '"$SYSDIR\tasklist.exe" /FI "IMAGENAME eq ${APP_EXE}" /FO CSV /NH'
  Pop $R0
  Pop $R1
  StrCmp $R0 "0" 0 desktopkit_install_process_ok
  StrLen $R2 '"${APP_EXE}"'
  StrCpy $R3 $R1 $R2
  StrCmp $R3 '"${APP_EXE}"' 0 desktopkit_install_process_ok
  IfSilent desktopkit_app_still_running
{{if .ConfirmStopRunning}}  MessageBox MB_YESNO|MB_ICONEXCLAMATION "$(DesktopKitConfirmStop)" IDYES desktopkit_confirm_stop
  SetErrorLevel 3
  Quit
 desktopkit_confirm_stop:
  ; The bundled helper only touches processes whose executable image matches
  ; the executable in $INSTDIR. It refuses unknown/unrelated same-name apps.
  InitPluginsDir
  SetOutPath "$PLUGINSDIR"
  File "/oname=kit-stop.ps1" "{{.ShutdownScript}}"
  nsExec::ExecToStack '"$SYSDIR\WindowsPowerShell\v1.0\powershell.exe" -NoLogo -NoProfile -NonInteractive -ExecutionPolicy Bypass -File "$PLUGINSDIR\kit-stop.ps1" -InstallDir "$INSTDIR" -ExeName "${APP_EXE}"'
  Pop $R0
  Pop $R1
  StrCmp $R0 "0" desktopkit_check_process
  MessageBox MB_OK|MB_ICONSTOP "$(DesktopKitStopFailed)"
  SetErrorLevel 4
  Quit
{{else}}  MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "$(DesktopKitCloseApp)" IDRETRY desktopkit_check_process
{{end}} desktopkit_app_still_running:
  SetErrorLevel 3
  Quit
 desktopkit_install_process_ok:
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

  {{if eq .InstallScope "user"}}StrCmp $DesktopKitRestartAfterInstall "1" 0 desktopkit_install_done
  Exec '"$INSTDIR\${APP_EXE}"'
desktopkit_install_done:{{end}}
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
