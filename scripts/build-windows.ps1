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
  Pop-Location
}

if (-not $SkipNative) {
  & (Join-Path $Root "native\windows\tsf\build.ps1") -Arch $NativeArch
}

Write-Host "Windows artifacts are under $(Join-Path $Root 'build\windows')"
