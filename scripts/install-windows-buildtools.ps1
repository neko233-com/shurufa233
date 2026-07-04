$ErrorActionPreference = "Stop"

winget install --id Microsoft.VisualStudio.2022.BuildTools -e `
  --accept-source-agreements `
  --accept-package-agreements `
  --override "--quiet --wait --norestart --add Microsoft.VisualStudio.Workload.VCTools --add Microsoft.VisualStudio.Component.VC.Tools.x86.x64 --add Microsoft.VisualStudio.Component.VC.Tools.ARM64 --add Microsoft.VisualStudio.Component.VC.Tools.ARM64EC --add Microsoft.VisualStudio.Component.Windows11SDK.26100 --includeRecommended"

$vswhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"
if (Test-Path $vswhere) {
  $installPath = & $vswhere -latest -products * -property installationPath
  $setup = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\setup.exe"
  if ((Test-Path $setup) -and $installPath) {
    & $setup modify --installPath $installPath --quiet --wait --norestart --includeRecommended `
      --add Microsoft.VisualStudio.Component.VC.Tools.x86.x64 `
      --add Microsoft.VisualStudio.Component.VC.Tools.ARM64 `
      --add Microsoft.VisualStudio.Component.VC.Tools.ARM64EC `
      --add Microsoft.VisualStudio.Component.Windows11SDK.26100
  }
}

Write-Host "Build Tools install/modify finished. Re-open PowerShell if cl.exe is not on PATH."
