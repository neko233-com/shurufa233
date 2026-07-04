param(
  [switch]$SkipBuild,
  [switch]$RegisterOnly,
  [switch]$ActivateProfile,
  [string]$TsfDllPath,
  [string]$CoreDllPath
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\.."
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\shurufa233"
$NativeInstallDir = Join-Path $env:ProgramFiles "shurufa233"
$ConfigDir = Join-Path $env:APPDATA "shurufa233"
$InputMethodBackupPath = Join-Path $ConfigDir "input-method-backup.json"
$RunKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
$Tip = "0804:{3D7B8D06-9872-4C31-B77D-3B87327CBF64}{B68911A2-4478-491C-A624-978441648E20}"

function Test-IsAdmin {
  $principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
  return $principal.IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator)
}

function Get-CurrentNativeArch {
  switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { return "x64" }
    "ARM64" { return "arm64" }
    "x86" { return "x86" }
    default { throw "Unsupported PROCESSOR_ARCHITECTURE=$env:PROCESSOR_ARCHITECTURE" }
  }
}

function Get-CurrentGoArch {
  switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { return "amd64" }
    "ARM64" { return "arm64" }
    "x86" { return "386" }
    default { throw "Unsupported PROCESSOR_ARCHITECTURE=$env:PROCESSOR_ARCHITECTURE" }
  }
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
      try { Remove-Item -LiteralPath $_.FullName -Force -ErrorAction Stop } catch {}
    }
  Get-ChildItem $NativeInstallDir -Filter "shurufa_core-*.dll" -ErrorAction SilentlyContinue |
    Where-Object { -not $KeepCoreDll -or $_.FullName -ne $KeepCoreDll } |
    ForEach-Object {
      try { Remove-Item -LiteralPath $_.FullName -Force -ErrorAction Stop } catch {}
    }
}

$NativeArch = Get-CurrentNativeArch
$GoArch = Get-CurrentGoArch

if (-not $SkipBuild -and -not $RegisterOnly) {
  & (Join-Path $Root "scripts\build-windows.ps1") -GoArch @($GoArch) -NativeArch @($NativeArch) -SkipFrontend
}

$DaemonSource = Join-Path $Root "build\windows\go-$GoArch\shurufa-daemon.exe"
$CliSource = Join-Path $Root "build\windows\go-$GoArch\shurufa-imecli.exe"
$CoreSource = Join-Path $Root "build\windows\go-$GoArch\shurufa_core.dll"
$TsfSource = Join-Path $Root "build\windows\$NativeArch\Shurufa233Tsf.dll"
$ProfileCtlSource = Join-Path $Root "build\windows\$NativeArch\Shurufa233ProfileCtl.exe"

if (-not $RegisterOnly) {
  foreach ($Path in @($DaemonSource, $CliSource, $TsfSource, $ProfileCtlSource)) {
    if (-not (Test-Path $Path)) {
      throw "Missing artifact: $Path"
    }
  }
} elseif (-not (Test-Path $TsfDllPath)) {
  throw "Missing TSF DLL for elevated registration: $TsfDllPath"
}

if (-not $RegisterOnly) {
  New-Item -ItemType Directory -Force $InstallDir | Out-Null
  Get-Process -Name shurufa-daemon -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
  for ($i = 0; $i -lt 20; $i++) {
    if (-not (Get-Process -Name shurufa-daemon -ErrorAction SilentlyContinue)) { break }
    Start-Sleep -Milliseconds 250
  }
  Copy-Item -Force $DaemonSource (Join-Path $InstallDir "shurufa-daemon.exe")
  Copy-Item -Force $CliSource (Join-Path $InstallDir "shurufa-imecli.exe")
  $stamp = Get-Date -Format "yyyyMMddHHmmss"
  $TsfDll = Join-Path $InstallDir "Shurufa233Tsf-$NativeArch-$stamp.dll"
  Copy-Item -Force $TsfSource $TsfDll
  if (Test-Path $CoreSource) {
    $LocalCoreDll = Join-Path $InstallDir "shurufa_core-$NativeArch-$stamp.dll"
    Copy-Item -Force $CoreSource $LocalCoreDll
    Copy-Item -Force $CoreSource (Join-Path $InstallDir "shurufa_core.dll")
  } else {
    Remove-Item -Force (Join-Path $InstallDir "shurufa_core.dll") -ErrorAction SilentlyContinue
    Write-Warning "shurufa_core.dll was not found for $GoArch; TSF will use daemon IPC fallback."
  }
  Copy-Item -Force $ProfileCtlSource (Join-Path $InstallDir "Shurufa233ProfileCtl.exe")

  $BundledDictionaryDir = Join-Path $Root "data\dictionaries"
  if (Test-Path $BundledDictionaryDir) {
    $UserDictionaryDir = Join-Path $ConfigDir "dictionaries"
    New-Item -ItemType Directory -Force $UserDictionaryDir | Out-Null
    Copy-Item -Force (Join-Path $BundledDictionaryDir "*.json") $UserDictionaryDir
  }
} else {
  $TsfDll = $TsfDllPath
  if ($CoreDllPath) {
    $LocalCoreDll = $CoreDllPath
  }
}

$Daemon = Join-Path $InstallDir "shurufa-daemon.exe"

New-Item -Path $RunKey -Force | Out-Null
Set-ItemProperty -Path $RunKey -Name "shurufa233-daemon" -Value "`"$Daemon`""

$existing = Get-Process shurufa-daemon -ErrorAction SilentlyContinue
if (-not $existing) {
  Start-Process -FilePath $Daemon -WorkingDirectory $InstallDir -WindowStyle Hidden
}

if (-not (Test-IsAdmin)) {
  $args = @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-File", "`"$PSCommandPath`"",
    "-SkipBuild",
    "-RegisterOnly",
    "-TsfDllPath", "`"$TsfDll`""
  )
  if ($LocalCoreDll -and (Test-Path $LocalCoreDll)) {
    $args += @("-CoreDllPath", "`"$LocalCoreDll`"")
  }
  $proc = Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -Wait -PassThru
  if ($proc.ExitCode -ne 0) {
    Write-Warning "Elevated TSF registration exited with code $($proc.ExitCode); verifying HKLM registration before failing."
  }
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
} else {
  New-Item -ItemType Directory -Force $NativeInstallDir | Out-Null
  $RegisteredTsfDll = Join-Path $NativeInstallDir (Split-Path $TsfDll -Leaf)
  Copy-Item -Force $TsfDll $RegisteredTsfDll
  if ($LocalCoreDll -and (Test-Path $LocalCoreDll)) {
    Copy-Item -Force $LocalCoreDll (Join-Path $NativeInstallDir (Split-Path $LocalCoreDll -Leaf))
    Copy-Item -Force $LocalCoreDll (Join-Path $NativeInstallDir "shurufa_core.dll") -ErrorAction SilentlyContinue
  } else {
    Remove-Item -Force (Join-Path $NativeInstallDir "shurufa_core.dll") -ErrorAction SilentlyContinue
  }
  regsvr32.exe /s $RegisteredTsfDll
  $RegisteredComPath = (Get-ItemProperty "HKLM:\Software\Classes\CLSID\{3D7B8D06-9872-4C31-B77D-3B87327CBF64}\InprocServer32")."(default)"
  if ($RegisteredComPath -ne $RegisteredTsfDll) {
    throw "TSF registration did not update HKLM. Expected $RegisteredTsfDll but found $RegisteredComPath"
  }
  $RegisteredCoreDll = if ($LocalCoreDll -and (Test-Path $LocalCoreDll)) {
    Join-Path $NativeInstallDir (Split-Path $LocalCoreDll -Leaf)
  } else {
    $null
  }
  Remove-StaleNativeArtifacts -KeepTsfDll $RegisteredTsfDll -KeepCoreDll $RegisteredCoreDll
}

if ($RegisterOnly) {
  Write-Host "Registered native TSF artifacts under $NativeInstallDir"
  return
}

if (-not $RegisterOnly) {
  Save-InputMethodBackup

  Get-ChildItem $InstallDir -Filter "Shurufa233Tsf-*.dll" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -ne $TsfDll } |
    ForEach-Object {
      try { Remove-Item -LiteralPath $_.FullName -Force -ErrorAction Stop } catch {}
    }
  Get-ChildItem $InstallDir -Filter "shurufa_core-*.dll" -ErrorAction SilentlyContinue |
    Where-Object { -not $LocalCoreDll -or $_.FullName -ne $LocalCoreDll } |
    ForEach-Object {
      try { Remove-Item -LiteralPath $_.FullName -Force -ErrorAction Stop } catch {}
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
Write-Host "Open Windows Settings > Time & language > Typing > Advanced keyboard settings to select shurufa233, or rerun with -ActivateProfile when testing."
