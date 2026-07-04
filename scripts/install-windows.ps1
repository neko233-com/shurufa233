param(
  [switch]$SkipBuild,
  [switch]$RegisterOnly,
  [switch]$ActivateProfile,
  [string]$TsfDllPath,
  [string]$CoreDllPath,
  [string]$CleanupReportPath
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\.."
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\shurufa233"
$NativeInstallDir = Join-Path $env:ProgramFiles "shurufa233"
$ConfigDir = Join-Path $env:APPDATA "shurufa233"
$InputMethodBackupPath = Join-Path $ConfigDir "input-method-backup.json"
$NativeCleanupReportPath = if ($CleanupReportPath) { $CleanupReportPath } else { Join-Path $ConfigDir "native-cleanup-report.json" }
$RunKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
$Tip = "0804:{3D7B8D06-9872-4C31-B77D-3B87327CBF64}{B68911A2-4478-491C-A624-978441648E20}"
$NativeCleanupStats = [ordered]@{
  attempted = 0
  removed = 0
  scheduledOnReboot = 0
  failed = 0
}

function Test-IsAdmin {
  $principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
  return $principal.IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator)
}

function Get-EffectiveProcessorArchitecture {
  if ($env:PROCESSOR_ARCHITEW6432) {
    return $env:PROCESSOR_ARCHITEW6432
  }
  return $env:PROCESSOR_ARCHITECTURE
}

function Get-CurrentNativeArch {
  $arch = Get-EffectiveProcessorArchitecture
  switch ($arch) {
    "AMD64" { return "x64" }
    "ARM64" { return "arm64" }
    "x86" { return "x86" }
    default { throw "Unsupported processor architecture. PROCESSOR_ARCHITECTURE=$env:PROCESSOR_ARCHITECTURE PROCESSOR_ARCHITEW6432=$env:PROCESSOR_ARCHITEW6432" }
  }
}

function Get-CurrentGoArch {
  $arch = Get-EffectiveProcessorArchitecture
  switch ($arch) {
    "AMD64" { return "amd64" }
    "ARM64" { return "arm64" }
    "x86" { return "386" }
    default { throw "Unsupported processor architecture. PROCESSOR_ARCHITECTURE=$env:PROCESSOR_ARCHITECTURE PROCESSOR_ARCHITEW6432=$env:PROCESSOR_ARCHITEW6432" }
  }
}

function Test-PackageManifest {
  param(
    [string]$ExpectedNativeArch,
    [string]$ExpectedGoArch
  )

  $manifestPath = Join-Path $Root "manifest.json"
  if (-not (Test-Path $manifestPath)) {
    return
  }

  $manifest = Get-Content -Path $manifestPath -Raw | ConvertFrom-Json
  if ($manifest.platform -ne "windows") {
    throw "Package manifest platform is '$($manifest.platform)', expected 'windows'."
  }
  if ($manifest.nativeArch -ne $ExpectedNativeArch -or $manifest.goArch -ne $ExpectedGoArch) {
    throw "This package targets nativeArch=$($manifest.nativeArch), goArch=$($manifest.goArch), but this Windows session needs nativeArch=$ExpectedNativeArch, goArch=$ExpectedGoArch. Use the matching shurufa233 Windows package."
  }
  if ($manifest.performanceMode -ne "in-process-core") {
    throw "Package performanceMode is '$($manifest.performanceMode)'. Production install requires in-process-core."
  }

  $requiredRoles = @("tsf-dll", "profilectl", "smokeedit", "daemon", "cli", "dictimport", "go-core", "settings-ui", "installer", "uninstaller")
  $presentRoles = @{}
  foreach ($artifact in @($manifest.artifacts)) {
    if ($artifact.required -eq $true -and $artifact.present -eq $true -and $artifact.role) {
      $presentRoles[[string]$artifact.role] = $true
    }
  }
  $missingRoles = @($requiredRoles | Where-Object { -not $presentRoles.ContainsKey($_) })
  if ($missingRoles.Count -gt 0) {
    throw "Package manifest is missing required production artifact role(s): $($missingRoles -join ', ')"
  }

  foreach ($artifact in @($manifest.artifacts)) {
    if ($artifact.required -ne $true) {
      continue
    }
    if ($artifact.present -ne $true) {
      throw "Package manifest marks required artifact as not present: $($artifact.path)"
    }
    $relativePath = [string]$artifact.path
    if ([IO.Path]::IsPathRooted($relativePath) -or $relativePath -match '(^|[\\/])\.\.([\\/]|$)') {
      throw "Package manifest contains an unsafe artifact path: $relativePath"
    }
    $fullPath = Join-Path $Root $relativePath
    if (-not (Test-Path $fullPath)) {
      throw "Package manifest required artifact is missing: $relativePath"
    }
    if ($artifact.sha256) {
      $actualHash = (Get-FileHash $fullPath -Algorithm SHA256).Hash
      if ($actualHash -ne $artifact.sha256) {
        throw "Package manifest hash mismatch for $relativePath. Expected $($artifact.sha256), found $actualHash"
      }
    }
  }
  Write-Host "Package manifest verified for $ExpectedNativeArch/$ExpectedGoArch."
}

function Save-InputMethodBackup {
  if (Test-Path $InputMethodBackupPath) {
    Write-Host "Input method backup already exists at $InputMethodBackupPath"
    return
  }

  New-Item -ItemType Directory -Force $ConfigDir | Out-Null
  $defaultOverride = Get-WinDefaultInputMethodOverride -ErrorAction SilentlyContinue
  $languages = @(
    Get-WinUserLanguageList | ForEach-Object {
      [pscustomobject]@{
        LanguageTag = $_.LanguageTag
        InputMethodTips = @($_.InputMethodTips)
      }
    }
  )
  $backup = [pscustomobject]@{
    Version = 1
    CreatedAt = (Get-Date).ToString("o")
    DefaultInputMethodTip = $defaultOverride.InputMethodTip
    Languages = $languages
  }
  $backup | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 $InputMethodBackupPath
  Write-Host "Saved input method backup to $InputMethodBackupPath"
}

function Remove-StaleNativeArtifacts {
  param(
    [string]$KeepTsfDll,
    [string]$KeepCoreDll
  )

  if (-not (Test-Path $NativeInstallDir)) {
    return
  }

  Get-ChildItem $NativeInstallDir -Filter "Shurufa233Tsf-*.dll" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -ne $KeepTsfDll } |
    ForEach-Object {
      Remove-OrScheduleFile -Path $_.FullName
    }
  Get-ChildItem $NativeInstallDir -Filter "shurufa_core-*.dll" -ErrorAction SilentlyContinue |
    Where-Object { -not $KeepCoreDll -or $_.FullName -ne $KeepCoreDll } |
    ForEach-Object {
      Remove-OrScheduleFile -Path $_.FullName
    }
}

function Initialize-NativeCleanupApi {
  if ("Shurufa233.NativeCleanup" -as [type]) {
    return
  }
  Add-Type -TypeDefinition @"
namespace Shurufa233 {
  using System;
  using System.Runtime.InteropServices;

  public static class NativeCleanup {
    [DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
    public static extern bool MoveFileEx(string existingFileName, string newFileName, int flags);
  }
}
"@
}

function Write-NativeCleanupReport {
  param(
    [string]$Phase,
    [string]$KeepTsfDll,
    [string]$KeepCoreDll
  )

  try {
    New-Item -ItemType Directory -Force $ConfigDir | Out-Null
    [pscustomobject]@{
      generatedAt = (Get-Date).ToString("o")
      phase = $Phase
      isAdmin = Test-IsAdmin
      nativeInstallDir = $NativeInstallDir
      keepTsfDll = $KeepTsfDll
      keepCoreDll = $KeepCoreDll
      attempted = $NativeCleanupStats["attempted"]
      removed = $NativeCleanupStats["removed"]
      scheduledOnReboot = $NativeCleanupStats["scheduledOnReboot"]
      failed = $NativeCleanupStats["failed"]
    } | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 $NativeCleanupReportPath
  } catch {}
}

function Write-NativeCleanupSummary {
  if (-not (Test-Path $NativeCleanupReportPath)) {
    return
  }
  try {
    $report = Get-Content $NativeCleanupReportPath -Raw | ConvertFrom-Json
    if ($report.attempted -gt 0) {
      Write-Host "Native cleanup: removed $($report.removed) stale DLL(s), scheduled $($report.scheduledOnReboot) locked DLL(s) for deletion on reboot, kept $($report.failed) locked DLL(s)."
    }
  } catch {}
}

function Remove-OrScheduleFile {
  param([string]$Path)

  if (-not (Test-Path $Path)) {
    return
  }
  $NativeCleanupStats["attempted"]++
  try {
    Remove-Item -LiteralPath $Path -Force -ErrorAction Stop
    $NativeCleanupStats["removed"]++
    Write-Verbose "Removed stale native artifact $Path"
    return
  } catch {
    try {
      Initialize-NativeCleanupApi
      $deleteOnReboot = 0x4
      if ([Shurufa233.NativeCleanup]::MoveFileEx($Path, $null, $deleteOnReboot)) {
        $NativeCleanupStats["scheduledOnReboot"]++
        Write-Verbose "Scheduled stale native artifact for deletion on reboot: $Path"
        return
      }
    } catch {}
    $NativeCleanupStats["failed"]++
    Write-Verbose "Could not remove stale native artifact $Path; it may still be locked by Windows."
  }
}

function Copy-OptionalFile {
  param(
    [string]$Source,
    [string]$Destination
  )

  try {
    Copy-Item -Force $Source $Destination -ErrorAction Stop
  } catch {
    Write-Verbose "Could not update optional compatibility file $Destination; it may be locked by Windows."
    Remove-OrScheduleFile -Path $Destination
  }
}

function Copy-VersionedTool {
  param(
    [string]$Source,
    [string]$InstallDir,
    [string]$BaseName,
    [string]$Stamp
  )

  $fixedPath = Join-Path $InstallDir "$BaseName.exe"
  try {
    Copy-Item -Force $Source $fixedPath -ErrorAction Stop
    return $fixedPath
  } catch {
    $versionedPath = Join-Path $InstallDir "$BaseName-$Stamp.exe"
    Copy-Item -Force $Source $versionedPath
    Write-Warning "Could not update $fixedPath because it is locked; installed latest tool at $versionedPath."
    return $versionedPath
  }
}

function Test-DaemonHealth {
  try {
    $response = Invoke-RestMethod -Uri "http://127.0.0.1:23333/health" -TimeoutSec 2 -ErrorAction Stop
    return $response.ok -eq $true
  } catch {
    return $false
  }
}

function Write-DaemonDiagnostics {
  $logPath = Join-Path $env:LOCALAPPDATA "shurufa233-daemon.log"
  if (Test-Path $logPath) {
    Write-Warning "shurufa-daemon did not become healthy. Recent daemon log from ${logPath}:"
    Get-Content -Path $logPath -Tail 80 -ErrorAction SilentlyContinue | ForEach-Object {
      Write-Warning $_
    }
  } else {
    Write-Warning "shurufa-daemon log was not found at $logPath"
  }
}

function Stop-TextServiceHost {
  Get-Process -Name ctfmon -ErrorAction SilentlyContinue |
    Stop-Process -Force -ErrorAction SilentlyContinue
  for ($i = 0; $i -lt 20; $i++) {
    if (-not (Get-Process -Name ctfmon -ErrorAction SilentlyContinue)) { break }
    Start-Sleep -Milliseconds 250
  }
}

function Stop-SmokeEditLabs {
  $processes = @(Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.ProcessName -like "Shurufa233SmokeEdit*" })
  if ($processes.Count -eq 0) {
    return
  }

  $processes | Stop-Process -Force -ErrorAction SilentlyContinue
  for ($i = 0; $i -lt 12; $i++) {
    $remaining = @(Get-Process -ErrorAction SilentlyContinue |
      Where-Object { $_.ProcessName -like "Shurufa233SmokeEdit*" })
    if ($remaining.Count -eq 0) {
      return
    }
    Start-Sleep -Milliseconds 250
  }

  $remaining = @(Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.ProcessName -like "Shurufa233SmokeEdit*" })
  foreach ($process in $remaining) {
    & cmd.exe /c "taskkill /F /PID $($process.Id) /T >nul 2>nul" | Out-Null
  }
  for ($i = 0; $i -lt 8; $i++) {
    if (-not (Get-Process -ErrorAction SilentlyContinue |
      Where-Object { $_.ProcessName -like "Shurufa233SmokeEdit*" })) {
      return
    }
    Start-Sleep -Milliseconds 250
  }
}

function Get-StartMenuDir {
  return Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\shurufa233"
}

function New-ShurufaShortcut {
  param(
    [string]$Path,
    [string]$TargetPath,
    [string]$Arguments,
    [string]$IconLocation,
    [string]$WorkingDirectory
  )

  $shell = New-Object -ComObject WScript.Shell
  $shortcut = $shell.CreateShortcut($Path)
  $shortcut.TargetPath = $TargetPath
  if ($Arguments) {
    $shortcut.Arguments = $Arguments
  }
  if ($IconLocation -and (Test-Path $IconLocation)) {
    $shortcut.IconLocation = $IconLocation
  }
  if ($WorkingDirectory -and (Test-Path $WorkingDirectory)) {
    $shortcut.WorkingDirectory = $WorkingDirectory
  }
  $shortcut.Save()
}

function Install-StartMenuShortcuts {
  param(
    [string]$InstallDir,
    [string]$SmokeEditPath
  )

  try {
    $startMenuDir = Get-StartMenuDir
    New-Item -ItemType Directory -Force $startMenuDir | Out-Null

    $settingsIcon = Join-Path $InstallDir "Shurufa233ProfileCtl.exe"
    New-ShurufaShortcut `
      -Path (Join-Path $startMenuDir "Settings.lnk") `
      -TargetPath (Join-Path $env:WINDIR "explorer.exe") `
      -Arguments "http://127.0.0.1:23333/settings/" `
      -IconLocation $settingsIcon `
      -WorkingDirectory $InstallDir

    $smokeEdit = if ($SmokeEditPath) { $SmokeEditPath } else { Join-Path $InstallDir "Shurufa233SmokeEdit.exe" }
    New-ShurufaShortcut `
      -Path (Join-Path $startMenuDir "Input Performance Lab.lnk") `
      -TargetPath $smokeEdit `
      -IconLocation $smokeEdit `
      -WorkingDirectory $InstallDir
  } catch {
    Write-Warning "Could not create Start Menu shortcuts: $($_.Exception.Message)"
  }
}

function ConvertTo-PowerShellLiteral {
  param([string]$Value)
  return "'" + ($Value -replace "'", "''") + "'"
}

function New-ElevatedRegisterArguments {
  $scriptPath = ConvertTo-PowerShellLiteral $PSCommandPath
  $tsfPath = ConvertTo-PowerShellLiteral $TsfDll
  $cleanupReportPath = ConvertTo-PowerShellLiteral $NativeCleanupReportPath
  $command = @"
`$ErrorActionPreference = 'Stop'
try {
  & $scriptPath -SkipBuild -RegisterOnly -TsfDllPath $tsfPath -CleanupReportPath $cleanupReportPath
  exit 0
} catch {
  [pscustomobject]@{
    generatedAt = (Get-Date).ToString('o')
    phase = 'register-only-error'
    error = (`$_ | Out-String)
  } | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 $cleanupReportPath
  Write-Error `$_
  exit 1
}
"@
  if ($LocalCoreDll -and (Test-Path $LocalCoreDll)) {
    $corePath = ConvertTo-PowerShellLiteral $LocalCoreDll
    $command = @"
`$ErrorActionPreference = 'Stop'
try {
  & $scriptPath -SkipBuild -RegisterOnly -TsfDllPath $tsfPath -CoreDllPath $corePath -CleanupReportPath $cleanupReportPath
  exit 0
} catch {
  [pscustomobject]@{
    generatedAt = (Get-Date).ToString('o')
    phase = 'register-only-error'
    error = (`$_ | Out-String)
  } | ConvertTo-Json -Depth 4 | Set-Content -Encoding UTF8 $cleanupReportPath
  Write-Error `$_
  exit 1
}
"@
  }
  $encoded = [Convert]::ToBase64String([Text.Encoding]::Unicode.GetBytes($command))
  return @("-NoProfile", "-ExecutionPolicy", "Bypass", "-EncodedCommand", $encoded)
}

function Start-DaemonAndWait {
  param([string]$DaemonPath)

  Get-Process -Name shurufa-daemon -ErrorAction SilentlyContinue |
    Stop-Process -Force -ErrorAction SilentlyContinue
  for ($i = 0; $i -lt 20; $i++) {
    if (-not (Get-Process -Name shurufa-daemon -ErrorAction SilentlyContinue)) { break }
    Start-Sleep -Milliseconds 250
  }

  Start-Process -FilePath $DaemonPath -WorkingDirectory (Split-Path $DaemonPath -Parent) -WindowStyle Hidden
  for ($i = 0; $i -lt 30; $i++) {
    if (Test-DaemonHealth) {
      return
    }
    Start-Sleep -Milliseconds 250
  }
  Write-DaemonDiagnostics
  throw "shurufa-daemon did not become healthy on http://127.0.0.1:23333/health"
}

function Register-NativeArtifacts {
  param(
    [string]$SourceTsfDll,
    [string]$SourceCoreDll
  )

  if (-not (Test-IsAdmin)) {
    throw "Register-NativeArtifacts requires an elevated PowerShell session."
  }

  New-Item -ItemType Directory -Force $NativeInstallDir | Out-Null
  $RegisteredTsfDll = Join-Path $NativeInstallDir (Split-Path $SourceTsfDll -Leaf)
  Copy-Item -Force $SourceTsfDll $RegisteredTsfDll
  if ($SourceCoreDll -and (Test-Path $SourceCoreDll)) {
    Copy-Item -Force $SourceCoreDll (Join-Path $NativeInstallDir (Split-Path $SourceCoreDll -Leaf))
    Copy-OptionalFile -Source $SourceCoreDll -Destination (Join-Path $NativeInstallDir "shurufa_core.dll")
  } else {
    Remove-OrScheduleFile -Path (Join-Path $NativeInstallDir "shurufa_core.dll")
  }
  Write-NativeCleanupReport -Phase "native-copied" -KeepTsfDll $RegisteredTsfDll -KeepCoreDll $SourceCoreDll

  $regsvr = Start-Process -FilePath "regsvr32.exe" -ArgumentList @("/s", "`"$RegisteredTsfDll`"") -Wait -PassThru
  $RegisteredComPath = (Get-ItemProperty "HKLM:\Software\Classes\CLSID\{3D7B8D06-9872-4C31-B77D-3B87327CBF64}\InprocServer32")."(default)"
  if ($RegisteredComPath -ne $RegisteredTsfDll) {
    throw "TSF registration did not update HKLM. Expected $RegisteredTsfDll but found $RegisteredComPath"
  }
  if ($regsvr.ExitCode -ne 0) {
    Write-Verbose "regsvr32 exited with code $($regsvr.ExitCode), but HKLM registration verification passed."
  }

  $RegisteredCoreDll = if ($SourceCoreDll -and (Test-Path $SourceCoreDll)) {
    Join-Path $NativeInstallDir (Split-Path $SourceCoreDll -Leaf)
  } else {
    $null
  }
  Remove-StaleNativeArtifacts -KeepTsfDll $RegisteredTsfDll -KeepCoreDll $RegisteredCoreDll
  Write-NativeCleanupReport -Phase "native-cleaned" -KeepTsfDll $RegisteredTsfDll -KeepCoreDll $RegisteredCoreDll
}

$NativeArch = Get-CurrentNativeArch
$GoArch = Get-CurrentGoArch

if (-not $RegisterOnly) {
  Test-PackageManifest -ExpectedNativeArch $NativeArch -ExpectedGoArch $GoArch
}

if (-not $SkipBuild -and -not $RegisterOnly) {
  & (Join-Path $Root "scripts\build-windows.ps1") -GoArch @($GoArch) -NativeArch @($NativeArch)
}

$DaemonSource = Join-Path $Root "build\windows\go-$GoArch\shurufa-daemon.exe"
$CliSource = Join-Path $Root "build\windows\go-$GoArch\shurufa-imecli.exe"
$DictImportSource = Join-Path $Root "build\windows\go-$GoArch\shurufa-dictimport.exe"
$CoreSource = Join-Path $Root "build\windows\go-$GoArch\shurufa_core.dll"
$TsfSource = Join-Path $Root "build\windows\$NativeArch\Shurufa233Tsf.dll"
$ProfileCtlSource = Join-Path $Root "build\windows\$NativeArch\Shurufa233ProfileCtl.exe"
$SmokeEditSource = Join-Path $Root "build\windows\$NativeArch\Shurufa233SmokeEdit.exe"
$SettingsSource = Join-Path $Root "apps\settings\dist"
$SmokeEditInstalledPath = Join-Path $InstallDir "Shurufa233SmokeEdit.exe"

if (-not $RegisterOnly) {
  foreach ($Path in @($DaemonSource, $CliSource, $DictImportSource, $TsfSource, $ProfileCtlSource, $SmokeEditSource)) {
    if (-not (Test-Path $Path)) {
      throw "Missing artifact: $Path"
    }
  }
} elseif (-not (Test-Path $TsfDllPath)) {
  throw "Missing TSF DLL for elevated registration: $TsfDllPath"
}

if (-not $RegisterOnly) {
  New-Item -ItemType Directory -Force $InstallDir | Out-Null
  Stop-TextServiceHost
  Get-Process -Name shurufa-daemon -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
  Stop-SmokeEditLabs
  for ($i = 0; $i -lt 20; $i++) {
    $daemonRunning = Get-Process -Name shurufa-daemon -ErrorAction SilentlyContinue
    $smokeRunning = Get-Process -ErrorAction SilentlyContinue |
      Where-Object { $_.ProcessName -like "Shurufa233SmokeEdit*" }
    if (-not $daemonRunning -and -not $smokeRunning) { break }
    Start-Sleep -Milliseconds 250
  }
  Copy-Item -Force $DaemonSource (Join-Path $InstallDir "shurufa-daemon.exe")
  Copy-Item -Force $CliSource (Join-Path $InstallDir "shurufa-imecli.exe")
  Copy-Item -Force $DictImportSource (Join-Path $InstallDir "shurufa-dictimport.exe")
  $stamp = Get-Date -Format "yyyyMMddHHmmss"
  $TsfDll = Join-Path $InstallDir "Shurufa233Tsf-$NativeArch-$stamp.dll"
  Copy-Item -Force $TsfSource $TsfDll
  if (Test-Path $CoreSource) {
    $LocalCoreDll = Join-Path $InstallDir "shurufa_core-$NativeArch-$stamp.dll"
    Copy-Item -Force $CoreSource $LocalCoreDll
    Copy-OptionalFile -Source $CoreSource -Destination (Join-Path $InstallDir "shurufa_core.dll")
  } else {
    Remove-OrScheduleFile -Path (Join-Path $InstallDir "shurufa_core.dll")
    Write-Warning "shurufa_core.dll was not found for $GoArch; TSF will use daemon IPC fallback."
  }
  Copy-Item -Force $ProfileCtlSource (Join-Path $InstallDir "Shurufa233ProfileCtl.exe")
  $SmokeEditInstalledPath = Copy-VersionedTool `
    -Source $SmokeEditSource `
    -InstallDir $InstallDir `
    -BaseName "Shurufa233SmokeEdit" `
    -Stamp $stamp
  Get-ChildItem -Path $InstallDir -Filter "Shurufa233SmokeEdit-*.exe" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -ne $SmokeEditInstalledPath } |
    ForEach-Object { Remove-OrScheduleFile -Path $_.FullName }
  if (Test-Path (Join-Path $SettingsSource "index.html")) {
    $SettingsInstallDir = Join-Path $InstallDir "settings"
    Remove-Item -LiteralPath $SettingsInstallDir -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force $SettingsInstallDir | Out-Null
    Copy-Item -Force -Recurse (Join-Path $SettingsSource "*") $SettingsInstallDir
  } else {
    Write-Warning "Settings UI build was not found at $SettingsSource; daemon /settings/ will be unavailable."
  }

  $BundledDictionaryDir = Join-Path $Root "data\dictionaries"
  if (Test-Path $BundledDictionaryDir) {
    $UserDictionaryDir = Join-Path $ConfigDir "dictionaries"
    New-Item -ItemType Directory -Force $UserDictionaryDir | Out-Null
    Copy-Item -Force (Join-Path $BundledDictionaryDir "*.json") $UserDictionaryDir
  }
  Install-StartMenuShortcuts -InstallDir $InstallDir -SmokeEditPath $SmokeEditInstalledPath
} else {
  $TsfDll = $TsfDllPath
  if ($CoreDllPath) {
    $LocalCoreDll = $CoreDllPath
  }
  Write-NativeCleanupReport -Phase "register-only-start" -KeepTsfDll $TsfDll -KeepCoreDll $LocalCoreDll
}

if ($RegisterOnly) {
  Register-NativeArtifacts -SourceTsfDll $TsfDll -SourceCoreDll $LocalCoreDll
  Write-Host "Registered native TSF artifacts under $NativeInstallDir"
  exit 0
}

$Daemon = Join-Path $InstallDir "shurufa-daemon.exe"

New-Item -Path $RunKey -Force | Out-Null
Set-ItemProperty -Path $RunKey -Name "shurufa233-daemon" -Value "`"$Daemon`""

if (-not $RegisterOnly) {
  Start-DaemonAndWait -DaemonPath $Daemon
}

if (-not (Test-IsAdmin)) {
  Remove-Item -LiteralPath $NativeCleanupReportPath -Force -ErrorAction SilentlyContinue
  $args = New-ElevatedRegisterArguments
  $proc = Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -Wait -PassThru
  $ElevatedExitCode = $proc.ExitCode
  Write-NativeCleanupSummary
  $ExpectedRegisteredTsfDll = Join-Path $NativeInstallDir (Split-Path $TsfDll -Leaf)
  $RegisteredComPath = (Get-ItemProperty "HKLM:\Software\Classes\CLSID\{3D7B8D06-9872-4C31-B77D-3B87327CBF64}\InprocServer32")."(default)"
  if ($RegisteredComPath -ne $ExpectedRegisteredTsfDll) {
    if (Test-Path $ExpectedRegisteredTsfDll) {
      Start-Process -FilePath "regsvr32.exe" -ArgumentList @("/s", "`"$ExpectedRegisteredTsfDll`"") -Verb RunAs -Wait
      $RegisteredComPath = (Get-ItemProperty "HKLM:\Software\Classes\CLSID\{3D7B8D06-9872-4C31-B77D-3B87327CBF64}\InprocServer32")."(default)"
    }
    if ($RegisteredComPath -ne $ExpectedRegisteredTsfDll) {
      throw "TSF registration did not update HKLM. Expected $ExpectedRegisteredTsfDll but found $RegisteredComPath"
    }
  }
  if ($LocalCoreDll -and (Test-Path $LocalCoreDll)) {
    $ExpectedCoreDll = Join-Path $NativeInstallDir (Split-Path $LocalCoreDll -Leaf)
    if (-not (Test-Path $ExpectedCoreDll)) {
      throw "Versioned core DLL was not installed to $ExpectedCoreDll"
    }
    $sourceHash = (Get-FileHash $LocalCoreDll -Algorithm SHA256).Hash
    $installedHash = (Get-FileHash $ExpectedCoreDll -Algorithm SHA256).Hash
    if ($sourceHash -ne $installedHash) {
      throw "Versioned core DLL hash mismatch. Expected $sourceHash but found $installedHash at $ExpectedCoreDll"
    }
  }
  if ($ElevatedExitCode -ne 0) {
    Write-Verbose "Elevated TSF registration exited with code $ElevatedExitCode, but HKLM registration and core DLL verification passed."
  }
} else {
  Register-NativeArtifacts -SourceTsfDll $TsfDll -SourceCoreDll $LocalCoreDll
}

if (-not $RegisterOnly) {
  Save-InputMethodBackup

  Get-ChildItem $InstallDir -Filter "Shurufa233Tsf-*.dll" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -ne $TsfDll } |
    ForEach-Object {
      Remove-OrScheduleFile -Path $_.FullName
    }
  Get-ChildItem $InstallDir -Filter "shurufa_core-*.dll" -ErrorAction SilentlyContinue |
    Where-Object { -not $LocalCoreDll -or $_.FullName -ne $LocalCoreDll } |
    ForEach-Object {
      Remove-OrScheduleFile -Path $_.FullName
    }
}

$languages = Get-WinUserLanguageList
$zh = $languages | Where-Object LanguageTag -eq "zh-Hans-CN" | Select-Object -First 1
if (-not $zh) {
  $languages = New-WinUserLanguageList zh-Hans-CN
  $zh = $languages[0]
}
if ($zh.InputMethodTips -notcontains $Tip) {
  $zh.InputMethodTips.Add($Tip)
  Set-WinUserLanguageList $languages -Force
}

Start-Process ctfmon.exe -WindowStyle Hidden -ErrorAction SilentlyContinue

$ProfileCtl = Join-Path $InstallDir "Shurufa233ProfileCtl.exe"
if (Test-Path $ProfileCtl) {
  & $ProfileCtl enable | Write-Host
  if ($ActivateProfile) {
    Set-WinDefaultInputMethodOverride -InputTip $Tip
    & $ProfileCtl activate | Write-Host
  }
}

Write-Host "Installed shurufa233 to $InstallDir"
Write-Host "Registered $NativeArch TSF DLL for the current user."
Write-Host "Daemon is configured for startup through HKCU Run."
Write-Host "Settings UI is served at http://127.0.0.1:23333/settings/."
Write-Host "Start Menu shortcuts are installed under shurufa233."
Write-Host "Input performance lab installed at $SmokeEditInstalledPath."
Write-Host "Open Windows Settings > Time & language > Typing > Advanced keyboard settings to select shurufa233, or rerun with -ActivateProfile when testing."
