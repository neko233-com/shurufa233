param(
  [ValidateSet("x64", "x86", "arm64")]
  [string[]]$Arch = @("x64", "arm64", "x86")
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\..\..\.."
$VsWhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"

function Get-VcVarsAll {
  if (Test-Path $VsWhere) {
    $installPath = & $VsWhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if ($installPath) {
      $candidate = Join-Path $installPath "VC\Auxiliary\Build\vcvarsall.bat"
      if (Test-Path $candidate) {
        return $candidate
      }
    }
  }
  throw "Visual Studio Build Tools with VC tools was not found. Install Microsoft.VisualStudio.2022.BuildTools with the VCTools workload."
}

function Invoke-VcBuild {
  param(
    [string]$VcVarsAll,
    [string]$TargetArch,
    [string]$Command
  )
  $cmd = "`"$VcVarsAll`" $TargetArch >nul && $Command"
  & cmd.exe /c $cmd
  if ($LASTEXITCODE -ne 0) {
    throw "cl.exe build failed for $TargetArch with exit code $LASTEXITCODE"
  }
}

function Get-VcTarget {
  param([string]$TargetArch)
  switch ($TargetArch) {
    "x64" { return "x64" }
    "arm64" { return "x64_arm64" }
    "x86" { return "x64_x86" }
    default { throw "Unsupported native arch $TargetArch" }
  }
}

$VcVarsAll = Get-VcVarsAll
$Source = Join-Path $PSScriptRoot "Shurufa233Tsf.cpp"
$Def = Join-Path $PSScriptRoot "Shurufa233Tsf.def"

foreach ($TargetArch in $Arch) {
  $Out = Join-Path $Root "build\windows\$TargetArch"
  New-Item -ItemType Directory -Force $Out | Out-Null
  $ObjDir = Join-Path $Out "obj"
  New-Item -ItemType Directory -Force $ObjDir | Out-Null

  $Dll = Join-Path $Out "Shurufa233Tsf.dll"
  $Obj = Join-Path $ObjDir "Shurufa233Tsf.obj"
  Remove-Item -Force $Dll -ErrorAction SilentlyContinue
  $compile = @(
    "cl.exe",
    "/nologo",
    "/std:c++20",
    "/EHsc",
    "/O2",
    "/GL",
    "/guard:cf",
    "/MD",
    "/DNDEBUG",
    "/DUNICODE",
    "/D_UNICODE",
    "/LD",
    "`"$Source`"",
    "/Fo`"$Obj`"",
    "/Fe:`"$Dll`"",
    "/link",
    "/DEF:`"$Def`"",
    "ole32.lib",
    "uuid.lib",
    "advapi32.lib",
    "winhttp.lib",
    "user32.lib",
    "gdi32.lib",
    "/OPT:REF",
    "/OPT:ICF",
    "/LTCG"
  ) -join " "

  Invoke-VcBuild -VcVarsAll $VcVarsAll -TargetArch (Get-VcTarget $TargetArch) -Command $compile
  if (-not (Test-Path $Dll)) {
    throw "Native build failed: $Dll was not created."
  }
  Write-Host "Built $Dll"
}
