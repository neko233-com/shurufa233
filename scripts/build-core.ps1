$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\.."
$Out = Join-Path $Root "build"
New-Item -ItemType Directory -Force $Out | Out-Null

Push-Location $Root
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

go build -o "$Out\shurufa-daemon.exe" .\cmd\daemon
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

go build -o "$Out\shurufa-imecli.exe" .\cmd\imecli
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$cgo = go env CGO_ENABLED
if ($cgo -eq "1") {
  go build -buildmode=c-shared -o "$Out\shurufa_core.dll" .\core\abi
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} else {
  Write-Warning "Skipping C ABI DLL because CGO_ENABLED=$cgo. Install a C/C++ toolchain and set CGO_ENABLED=1 to build core/abi."
}
Pop-Location

Write-Host "Built Go engine, daemon, and CLI under $Out"
