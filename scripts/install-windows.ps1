$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\.."

Write-Host "Checking Windows native toolchain..."
$cl = Get-Command cl.exe -ErrorAction SilentlyContinue
if (-not $cl) {
  Write-Host "Missing cl.exe. Install Visual Studio Build Tools with the Windows SDK."
  Write-Host "The Go engine, daemon, and settings UI can still run now."
  exit 2
}

& "$Root\native\windows\tsf\build.ps1"

$TsfDll = Join-Path $Root "build\windows\Shurufa233Tsf.dll"
if (-not (Test-Path $TsfDll)) {
  throw "TSF DLL build failed: $TsfDll was not created."
}

regsvr32.exe /s $TsfDll
Write-Host "Registered shurufa233 TSF COM server for the current user."
Write-Host "Next step: implement ITfInputProcessorProfile registration and enable the profile in Windows language settings."
