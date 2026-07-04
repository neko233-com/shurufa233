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
    "--wait",
    "--norestart",
    "--includeRecommended",
    "--add", "Microsoft.VisualStudio.Component.VC.Tools.x86.x64",
    "--add", "Microsoft.VisualStudio.Component.VC.14.44.17.14.ARM64",
    "--add", "Microsoft.VisualStudio.Component.VC.Tools.ARM64",
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

winget install --id Microsoft.VisualStudio.2022.BuildTools -e `
  --accept-source-agreements `
  --accept-package-agreements `
  --override "--quiet --wait --norestart --add Microsoft.VisualStudio.Workload.VCTools --add Microsoft.VisualStudio.Component.VC.Tools.x86.x64 --add Microsoft.VisualStudio.Component.VC.14.44.17.14.ARM64 --add Microsoft.VisualStudio.Component.VC.Tools.ARM64 --add Microsoft.VisualStudio.Component.VC.Tools.ARM64EC --add Microsoft.VisualStudio.Component.Windows11SDK.26100 --includeRecommended"

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

Write-Host "Build Tools install/modify finished and MSVC x64/x86/arm64 runtime libraries were found. Re-open PowerShell if cl.exe is not on PATH."
