$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\.."
$Out = Join-Path $Root "build"
New-Item -ItemType Directory -Force $Out | Out-Null

function Add-MingwToPath {
  $gcc = Get-Command gcc.exe -ErrorAction SilentlyContinue
  if ($gcc) {
    return $true
  }

  $packageRoot = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
  if (Test-Path $packageRoot) {
    $winlibs = Get-ChildItem $packageRoot -Directory -ErrorAction SilentlyContinue |
      Where-Object { $_.Name -like "BrechtSanders.WinLibs.POSIX.UCRT*" } |
      Select-Object -First 1
    if ($winlibs) {
      $bin = Join-Path $winlibs.FullName "mingw64\bin"
      if (Test-Path (Join-Path $bin "gcc.exe")) {
        $env:Path = "$bin;$env:Path"
        return $true
      }
    }
  }

  return $false
}

Push-Location $Root
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

go build -o "$Out\shurufa-daemon.exe" .\cmd\daemon
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

go build -o "$Out\shurufa-imecli.exe" .\cmd\imecli
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$hasMingw = Add-MingwToPath
if ($hasMingw) {
  $env:CGO_ENABLED = "1"
  $env:CC = "gcc"
  $env:CXX = "g++"
}

$cgo = go env CGO_ENABLED
if ($cgo -eq "1" -and $hasMingw) {
  go build -buildmode=c-shared -o "$Out\shurufa_core.dll" .\core\abi
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} else {
  Write-Warning "Skipping C ABI DLL because MinGW-w64 GCC was not found. Install BrechtSanders.WinLibs.POSIX.UCRT with winget to build core/abi."
}
Pop-Location

Write-Host "Built Go engine, daemon, and CLI under $Out"
