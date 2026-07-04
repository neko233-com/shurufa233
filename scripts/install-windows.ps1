param(
  [switch]$SkipBuild,
  [switch]$RegisterOnly,
  [string]$TsfDllPath,
  [string]$CoreDllPath
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\.."
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\shurufa233"
$NativeInstallDir = Join-Path $env:ProgramFiles "shurufa233"
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
  if (Test-Path $CoreSource) {
    $LocalCoreDll = Join-Path $InstallDir "shurufa_core.dll"
    Copy-Item -Force $CoreSource $LocalCoreDll
  } else {
    Remove-Item -Force (Join-Path $InstallDir "shurufa_core.dll") -ErrorAction SilentlyContinue
    Write-Warning "shurufa_core.dll was not found for $GoArch; TSF will use daemon IPC fallback."
  }
  Copy-Item -Force $ProfileCtlSource (Join-Path $InstallDir "Shurufa233ProfileCtl.exe")
  $stamp = Get-Date -Format "yyyyMMddHHmmss"
  $TsfDll = Join-Path $InstallDir "Shurufa233Tsf-$NativeArch-$stamp.dll"
  Copy-Item -Force $TsfSource $TsfDll
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
  Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -Wait
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
} else {
  New-Item -ItemType Directory -Force $NativeInstallDir | Out-Null
  $RegisteredTsfDll = Join-Path $NativeInstallDir (Split-Path $TsfDll -Leaf)
  Copy-Item -Force $TsfDll $RegisteredTsfDll
  if ($LocalCoreDll -and (Test-Path $LocalCoreDll)) {
    Copy-Item -Force $LocalCoreDll (Join-Path $NativeInstallDir "shurufa_core.dll")
  } else {
    Remove-Item -Force (Join-Path $NativeInstallDir "shurufa_core.dll") -ErrorAction SilentlyContinue
  }
  regsvr32.exe /s $RegisteredTsfDll
  $RegisteredComPath = (Get-ItemProperty "HKLM:\Software\Classes\CLSID\{3D7B8D06-9872-4C31-B77D-3B87327CBF64}\InprocServer32")."(default)"
  if ($RegisteredComPath -ne $RegisteredTsfDll) {
    throw "TSF registration did not update HKLM. Expected $RegisteredTsfDll but found $RegisteredComPath"
  }
}

if (-not $RegisterOnly) {
  Get-ChildItem $InstallDir -Filter "Shurufa233Tsf-*.dll" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -ne $TsfDll } |
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
Set-WinDefaultInputMethodOverride -InputTip $Tip

Start-Process ctfmon.exe -WindowStyle Hidden -ErrorAction SilentlyContinue

$ProfileCtl = Join-Path $InstallDir "Shurufa233ProfileCtl.exe"
if (Test-Path $ProfileCtl) {
  & $ProfileCtl enable | Write-Host
  & $ProfileCtl activate | Write-Host
}

Write-Host "Installed shurufa233 to $InstallDir"
Write-Host "Registered $NativeArch TSF DLL for the current user."
Write-Host "Daemon is configured for startup through HKCU Run."
Write-Host "Open Windows Settings > Time & language > Typing > Advanced keyboard settings to select shurufa233."
