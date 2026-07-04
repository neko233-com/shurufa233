$ErrorActionPreference = "Stop"

function Test-IsAdministrator {
  $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
  $principal = [Security.Principal.WindowsPrincipal]::new($identity)
  return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Invoke-VsSetupModify {
  param(
    [string]$Setup,
    [string]$InstallPath
  )

  $args = @(
    "modify",
    "--installPath", "`"$InstallPath`"",
    "--quiet",
    "--norestart",
    "--includeRecommended",
    "--add", "Microsoft.VisualStudio.Component.VC.Tools.x86.x64",
    "--add", "Microsoft.VisualStudio.Component.VC.14.44.17.14.ARM64",
    "--add", "Microsoft.VisualStudio.Component.VC.Tools.ARM64EC",
    "--add", "Microsoft.VisualStudio.Component.Windows11SDK.26100"
  )

  if (Test-IsAdministrator) {
    & $Setup @args
    if ($LASTEXITCODE -ne 0) {
      throw "Visual Studio setup modify failed with exit code $LASTEXITCODE."
    }
    return
  }

  Write-Host "Requesting UAC to modify Visual Studio Build Tools..."
  $proc = Start-Process -FilePath $Setup -ArgumentList $args -Verb RunAs -Wait -PassThru
  if ($proc.ExitCode -ne 0) {
    throw "Visual Studio setup modify failed with exit code $($proc.ExitCode)."
  }
}

function Test-MsvcRuntimeLibrary {
  param(
    [string]$InstallPath,
    [string]$Arch
  )

  $msvcRoot = Join-Path $InstallPath "VC\Tools\MSVC"
  if (-not (Test-Path $msvcRoot)) {
    return $false
  }

  $found = Get-ChildItem $msvcRoot -Recurse -Filter "msvcprt.lib" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -match "\\lib\\$Arch\\msvcprt\.lib$" } |
    Select-Object -First 1
  return [bool]$found
}

function Find-MingwCompiler {
  param([string]$Name)

  $found = Get-Command $Name -ErrorAction SilentlyContinue
  if ($found) {
    return $found.Source
  }

  $searchRoots = @()
  $wingetRoot = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
  if (Test-Path $wingetRoot) {
    $searchRoots += $wingetRoot
  }
  foreach ($parent in @($env:LOCALAPPDATA, $env:ProgramFiles, ${env:ProgramFiles(x86)})) {
    if (-not $parent -or -not (Test-Path $parent)) { continue }
    Get-ChildItem $parent -Directory -ErrorAction SilentlyContinue |
      Where-Object { $_.Name -match "mingw|llvm|winlibs|ucrt" } |
      ForEach-Object { $searchRoots += $_.FullName }
  }

  foreach ($root in $searchRoots) {
    $candidate = Get-ChildItem $root -Recurse -Filter $Name -File -ErrorAction SilentlyContinue |
      Where-Object { $_.FullName -match "\\bin\\[^\\]+$" } |
      Select-Object -First 1
    if ($candidate) {
      return $candidate.FullName
    }
  }

  return $null
}

winget install --id Microsoft.VisualStudio.2022.BuildTools -e `
  --accept-source-agreements `
  --accept-package-agreements `
  --override "--quiet --wait --norestart --add Microsoft.VisualStudio.Workload.VCTools --add Microsoft.VisualStudio.Component.VC.Tools.x86.x64 --add Microsoft.VisualStudio.Component.VC.14.44.17.14.ARM64 --add Microsoft.VisualStudio.Component.VC.Tools.ARM64EC --add Microsoft.VisualStudio.Component.Windows11SDK.26100 --includeRecommended"

winget install --id MartinStorsjo.LLVM-MinGW.UCRT -e `
  --accept-source-agreements `
  --accept-package-agreements

$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $vswhere) {
  $installPath = & $vswhere -latest -products * -property installationPath
  $setup = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\setup.exe"
  if ((Test-Path $setup) -and $installPath) {
    Invoke-VsSetupModify -Setup $setup -InstallPath $installPath

    $missing = @()
    foreach ($arch in @("x64", "x86", "arm64")) {
      if (-not (Test-MsvcRuntimeLibrary -InstallPath $installPath -Arch $arch)) {
        $missing += $arch
      }
    }

    if ($missing.Count -gt 0) {
      throw "Build Tools install/modify finished, but MSVC runtime libraries are still missing for: $($missing -join ', '). Open Visual Studio Installer, modify Build Tools, and enable MSVC ARM64/ARM64EC build tools plus Windows 11 SDK 26100."
    }
  }
}

$missingMingw = @()
foreach ($compiler in @("x86_64-w64-mingw32-gcc.exe", "i686-w64-mingw32-gcc.exe", "aarch64-w64-mingw32-gcc.exe")) {
  if (-not (Find-MingwCompiler -Name $compiler)) {
    $missingMingw += $compiler
  }
}
if ($missingMingw.Count -gt 0) {
  throw "LLVM-MinGW install finished, but these MinGW-w64 compilers were not found: $($missingMingw -join ', '). Re-open PowerShell or reinstall MartinStorsjo.LLVM-MinGW.UCRT."
}

Write-Host "Build Tools install/modify finished. MSVC x64/x86/arm64 runtime checks and LLVM-MinGW compiler checks passed. Re-open PowerShell if new tools are not on PATH."
