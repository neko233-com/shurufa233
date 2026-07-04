param(
  [ValidateSet("amd64", "386", "arm64")]
  [string[]]$GoArch = @("amd64", "arm64", "386"),

  [ValidateSet("x64", "x86", "arm64")]
  [string[]]$NativeArch = @("x64", "arm64", "x86"),

  [switch]$SkipFrontend,
  [switch]$SkipNative
)

$ErrorActionPreference = "Stop"
$Root = Resolve-Path "$PSScriptRoot\.."

function Find-MingwCompiler {
  param([string]$Arch)

  $preferred = switch ($Arch) {
    "amd64" { @("x86_64-w64-mingw32-gcc.exe", "gcc.exe") }
    "386" { @("i686-w64-mingw32-gcc.exe") }
    "arm64" { @("aarch64-w64-mingw32-gcc.exe") }
    default { @() }
  }

  foreach ($name in $preferred) {
    $found = Get-Command $name -ErrorAction SilentlyContinue
    if ($found) {
      return $found.Source
    }
  }

  $packageRoot = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
  if (Test-Path $packageRoot) {
    $winlibs = Get-ChildItem $packageRoot -Directory -ErrorAction SilentlyContinue |
      Where-Object { $_.Name -like "BrechtSanders.WinLibs.POSIX.UCRT*" } |
      Select-Object -First 1
    if ($winlibs) {
      $bin = Join-Path $winlibs.FullName "mingw64\bin"
      foreach ($name in $preferred) {
        $candidate = Join-Path $bin $name
        if (Test-Path $candidate) {
          $env:Path = "$bin;$env:Path"
          return $candidate
        }
      }
    }
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
    foreach ($name in $preferred) {
      $candidate = Get-ChildItem $root -Recurse -Filter $name -File -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -match "\\bin\\[^\\]+$" } |
        Select-Object -First 1
      if ($candidate) {
        $bin = Split-Path $candidate.FullName -Parent
        $env:Path = "$bin;$env:Path"
        return $candidate.FullName
      }
    }
  }

  return $null
}

if (-not $SkipFrontend) {
  Push-Location (Join-Path $Root "apps\settings")
  npm install
  npm run build
  Pop-Location
}

foreach ($Arch in $GoArch) {
  $Out = Join-Path $Root "build\windows\go-$Arch"
  New-Item -ItemType Directory -Force $Out | Out-Null
  Push-Location $Root
  $env:GOOS = "windows"
  $env:GOARCH = $Arch
  $env:CGO_ENABLED = "0"
  go build -trimpath -ldflags="-s -w" -o (Join-Path $Out "shurufa-daemon.exe") .\cmd\daemon
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
  go build -trimpath -ldflags="-s -w" -o (Join-Path $Out "shurufa-imecli.exe") .\cmd\imecli
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
  go build -trimpath -ldflags="-s -w" -o (Join-Path $Out "shurufa-dictimport.exe") .\cmd\dictimport
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
  go build -trimpath -ldflags="-s -w" -o (Join-Path $Out "shurufa-dictmanifest.exe") .\cmd\dictmanifest
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

  $compiler = Find-MingwCompiler -Arch $Arch
  if ($compiler) {
    $env:CGO_ENABLED = "1"
    $env:CC = $compiler
    $env:CXX = $compiler -replace "gcc(\.exe)?$", "g++`$1"
    go build -buildmode=c-shared -trimpath -ldflags="-s -w" -o (Join-Path $Out "shurufa_core.dll") .\core\abi
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
  } else {
    Write-Warning "Skipping shurufa_core.dll for GOARCH=$Arch because a matching MinGW-w64 compiler was not found."
  }
  Pop-Location
}

if (-not $SkipNative) {
  & (Join-Path $Root "native\windows\tsf\build.ps1") -Arch $NativeArch
}

Write-Host "Windows artifacts are under $(Join-Path $Root 'build\windows')"
