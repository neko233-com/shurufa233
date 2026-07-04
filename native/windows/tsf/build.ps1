$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\..\..\.."
$Out = Join-Path $Root "build\windows"
New-Item -ItemType Directory -Force $Out | Out-Null

if (-not (Get-Command cl.exe -ErrorAction SilentlyContinue)) {
  throw "cl.exe was not found. Install Visual Studio Build Tools with the Windows SDK, then run from a Developer PowerShell."
}

$cgo = go env CGO_ENABLED
if ($cgo -ne "1") {
  throw "CGO_ENABLED=$cgo. Run 'go env -w CGO_ENABLED=1' after installing a C/C++ toolchain."
}

Push-Location $Root
go build -buildmode=c-shared -o "$Out\shurufa_core.dll" .\core\abi
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Pop-Location

$Source = Join-Path $PSScriptRoot "Shurufa233Tsf.cpp"
$Def = Join-Path $PSScriptRoot "Shurufa233Tsf.def"
cl.exe /nologo /std:c++20 /EHsc /LD $Source /Fe:"$Out\Shurufa233Tsf.dll" /link /DEF:$Def msctf.lib ole32.lib advapi32.lib
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Built $Out\Shurufa233Tsf.dll"
