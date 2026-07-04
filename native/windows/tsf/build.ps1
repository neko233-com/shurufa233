param(
  [ValidateSet("x64", "x86", "arm64")]
  [string[]]$Arch = @("x64", "arm64", "x86")
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\..\..\.."
$VsWhere = "${env:ProgramFiles(x86)}\Microsoft Visual Studio\Installer\vswhere.exe"

function Get-VsInstallPath {
  if (Test-Path $VsWhere) {
    $installPath = & $VsWhere -latest -products * -requires Microsoft.VisualStudio.Component.VC.Tools.x86.x64 -property installationPath
    if ($installPath) {
      return $installPath
    }
  }
  throw "Visual Studio Build Tools with VC tools was not found. Install Microsoft.VisualStudio.2022.BuildTools with the VCTools workload."
}

function Get-VcVarsAll {
  param([string]$InstallPath)

  $candidate = Join-Path $InstallPath "VC\Auxiliary\Build\vcvarsall.bat"
  if (Test-Path $candidate) {
    return $candidate
  }
  throw "vcvarsall.bat was not found under $InstallPath."
}

function Get-VcLibArch {
  param([string]$TargetArch)

  switch ($TargetArch) {
    "x64" { return "x64" }
    "arm64" { return "arm64" }
    "x86" { return "x86" }
    default { throw "Unsupported native arch $TargetArch" }
  }
}

function Assert-VcRuntimeLibrary {
  param(
    [string]$InstallPath,
    [string]$TargetArch
  )

  $libArch = Get-VcLibArch $TargetArch
  $msvcRoot = Join-Path $InstallPath "VC\Tools\MSVC"
  $runtimeLib = Get-ChildItem $msvcRoot -Recurse -Filter "msvcprt.lib" -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -match "\\lib\\$libArch\\msvcprt\.lib$" } |
    Select-Object -First 1

  if (-not $runtimeLib) {
    $componentHint = if ($TargetArch -eq "arm64") {
      "Microsoft.VisualStudio.Component.VC.Tools.ARM64 and Microsoft.VisualStudio.Component.VC.14.44.17.14.ARM64"
    } else {
      "Microsoft.VisualStudio.Component.VC.Tools.x86.x64"
    }

    throw "MSVC runtime library for native arch '$TargetArch' was not found under '$msvcRoot'. Install/modify Visual Studio Build Tools with: $componentHint, then rerun scripts\install-windows-buildtools.ps1."
  }
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

$VsInstallPath = Get-VsInstallPath
$VcVarsAll = Get-VcVarsAll -InstallPath $VsInstallPath
$Source = Join-Path $PSScriptRoot "Shurufa233Tsf.cpp"
$Def = Join-Path $PSScriptRoot "Shurufa233Tsf.def"
$ProfileCtlSource = Join-Path $PSScriptRoot "..\profilectl\Shurufa233ProfileCtl.cpp"

foreach ($TargetArch in $Arch) {
  Assert-VcRuntimeLibrary -InstallPath $VsInstallPath -TargetArch $TargetArch

  $Out = Join-Path $Root "build\windows\$TargetArch"
  New-Item -ItemType Directory -Force $Out | Out-Null
  $ObjDir = Join-Path $Out "obj"
  New-Item -ItemType Directory -Force $ObjDir | Out-Null

  $Dll = Join-Path $Out "Shurufa233Tsf.dll"
  $Obj = Join-Path $ObjDir "Shurufa233Tsf.obj"
  $ProfileCtl = Join-Path $Out "Shurufa233ProfileCtl.exe"
  $ProfileCtlObj = Join-Path $ObjDir "Shurufa233ProfileCtl.obj"
  Remove-Item -Force $Dll -ErrorAction SilentlyContinue
  Remove-Item -Force $ProfileCtl -ErrorAction SilentlyContinue
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
    "dwmapi.lib",
    "/OPT:REF",
    "/OPT:ICF",
    "/LTCG"
  ) -join " "

  Invoke-VcBuild -VcVarsAll $VcVarsAll -TargetArch (Get-VcTarget $TargetArch) -Command $compile
  if (-not (Test-Path $Dll)) {
    throw "Native build failed: $Dll was not created."
  }
  Write-Host "Built $Dll"

  $profileCompile = @(
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
    "`"$ProfileCtlSource`"",
    "/Fo`"$ProfileCtlObj`"",
    "/Fe:`"$ProfileCtl`"",
    "/link",
    "ole32.lib",
    "uuid.lib",
    "/OPT:REF",
    "/OPT:ICF",
    "/LTCG"
  ) -join " "

  Invoke-VcBuild -VcVarsAll $VcVarsAll -TargetArch (Get-VcTarget $TargetArch) -Command $profileCompile
  if (-not (Test-Path $ProfileCtl)) {
    throw "Native build failed: $ProfileCtl was not created."
  }
  Write-Host "Built $ProfileCtl"
}
