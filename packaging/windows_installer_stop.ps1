param(
    [Parameter(Mandatory = $true)][string]$InstallDir,
    [Parameter(Mandatory = $true)][string]$ExeName
)

# This helper is unpacked by the installer and executed ONLY after its
# interactive close-and-continue confirmation. A silent Setup never calls it.
# Consent includes forced termination of the exact installed EXE (and any
# background instances launched from that same image).
$ErrorActionPreference = 'Stop'

try {
    if ($ExeName -notmatch '^[A-Za-z0-9_.-]+\.exe$') { exit 4 }
    $expectedPath = [IO.Path]::GetFullPath((Join-Path $InstallDir $ExeName))
    $filter = "Name = '" + $ExeName.Replace("'", "''") + "'"
    $matches = @(Get-CimInstance Win32_Process -Filter $filter -ErrorAction Stop)

    # Fail closed if any same-name image has an unknown or different path.
    # In particular, never terminate portable instances of another installation.
    foreach ($process in $matches) {
        if ([string]::IsNullOrWhiteSpace($process.ExecutablePath)) { exit 4 }
        $actual = [IO.Path]::GetFullPath($process.ExecutablePath)
        if (-not [string]::Equals($actual, $expectedPath, [StringComparison]::OrdinalIgnoreCase)) {
            exit 4
        }
    }

    foreach ($process in $matches) {
        # The installer warning explicitly discloses task interruption.
        # This is scoped to the exact installed executable, never arbitrary PIDs.
        Stop-Process -Id $process.ProcessId -Force -ErrorAction SilentlyContinue
    }

    $until = (Get-Date).AddSeconds(10)
    do {
        $remaining = @(Get-CimInstance Win32_Process -Filter $filter -ErrorAction Stop)
        if ($remaining.Count -eq 0) { exit 0 }
        Start-Sleep -Milliseconds 200
    } while ((Get-Date) -lt $until)
} catch {
    Write-Error $_
}
exit 4
