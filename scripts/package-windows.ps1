param(
  [ValidateSet("x64", "x86", "arm64")]
  [string[]]$Arch = @("x64", "arm64", "x86"),

  [string]$Version,
  [switch]$SkipMissingArch
)

$ErrorActionPreference = "Stop"

$Root = Resolve-Path "$PSScriptRoot\.."
$DistRoot = Join-Path $Root "dist\windows"

function Get-GoArch {
  param([string]$NativeArch)
  switch ($NativeArch) {
    "x64" { return "amd64" }
    "x86" { return "386" }
    "arm64" { return "arm64" }
    default { throw "Unsupported native arch $NativeArch" }
  }
}

function Get-GitCommit {
  try {
    $commit = (& git -C $Root rev-parse --short HEAD 2>$null).Trim()
    if ($commit) { return $commit }
  } catch {}
  return "nogit"
}

function Copy-RequiredFile {
  param(
    [string]$Source,
    [string]$Destination
  )
  if (-not (Test-Path $Source)) {
    throw "Missing required artifact: $Source"
  }
  New-Item -ItemType Directory -Force (Split-Path $Destination -Parent) | Out-Null
  Copy-Item -Force $Source $Destination
}

function Add-ArtifactHash {
  param(
    [System.Collections.Generic.List[object]]$Artifacts,
    [string]$Path,
    [string]$RelativePath,
    [string]$Role,
    [bool]$Required
  )

  if (-not (Test-Path $Path)) {
    $Artifacts.Add([pscustomobject]@{
      role = $Role
      path = $Path
      required = $Required
      present = $false
      sha256 = $null
      size = 0
    })
    return
  }

  $item = Get-Item $Path
  $hash = Get-FileHash $Path -Algorithm SHA256
  $Artifacts.Add([pscustomobject]@{
    role = $Role
    path = $RelativePath
    required = $Required
    present = $true
    sha256 = $hash.Hash
    size = $item.Length
  })
}

function Get-ArtifactRole {
  param(
    [string]$RelativePath,
    [string]$NativeArch,
    [string]$GoArch
  )

  $normalized = $RelativePath -replace '/', '\'
  switch -Regex ($normalized) {
    "^build\\windows\\$([regex]::Escape($NativeArch))\\Shurufa233Tsf\.dll$" { return "tsf-dll" }
    "^build\\windows\\$([regex]::Escape($NativeArch))\\Shurufa233ProfileCtl\.exe$" { return "profilectl" }
    "^build\\windows\\$([regex]::Escape($NativeArch))\\Shurufa233SmokeEdit\.exe$" { return "smokeedit" }
    "^build\\windows\\go-$([regex]::Escape($GoArch))\\shurufa-daemon\.exe$" { return "daemon" }
    "^build\\windows\\go-$([regex]::Escape($GoArch))\\shurufa-imecli\.exe$" { return "cli" }
    "^build\\windows\\go-$([regex]::Escape($GoArch))\\shurufa-dictimport\.exe$" { return "dictimport" }
    "^build\\windows\\go-$([regex]::Escape($GoArch))\\shurufa-dictmanifest\.exe$" { return "dictmanifest" }
    "^build\\windows\\go-$([regex]::Escape($GoArch))\\shurufa-dictsync\.exe$" { return "dictsync" }
    "^build\\windows\\go-$([regex]::Escape($GoArch))\\shurufa_core\.dll$" { return "go-core" }
    "^apps\\settings\\dist\\index\.html$" { return "settings-ui" }
    "^scripts\\install-windows\.ps1$" { return "installer" }
    "^scripts\\uninstall-windows\.ps1$" { return "uninstaller" }
    "^data\\dictionaries\\" { return "dictionary" }
    "^docs\\" { return "docs" }
    default { return "package-file" }
  }
}

if (-not $Version) {
  $Version = (Get-Date -Format "yyyyMMddHHmmss")
}
$GitCommit = Get-GitCommit

New-Item -ItemType Directory -Force $DistRoot | Out-Null

foreach ($NativeArch in $Arch) {
  $GoArch = Get-GoArch $NativeArch
  $NativeOut = Join-Path $Root "build\windows\$NativeArch"
  $GoOut = Join-Path $Root "build\windows\go-$GoArch"

  $required = @(
    (Join-Path $NativeOut "Shurufa233Tsf.dll"),
    (Join-Path $NativeOut "Shurufa233ProfileCtl.exe"),
    (Join-Path $NativeOut "Shurufa233SmokeEdit.exe"),
    (Join-Path $GoOut "shurufa-daemon.exe"),
    (Join-Path $GoOut "shurufa-imecli.exe"),
    (Join-Path $GoOut "shurufa-dictimport.exe"),
    (Join-Path $GoOut "shurufa-dictmanifest.exe"),
    (Join-Path $GoOut "shurufa-dictsync.exe")
  )
  $missing = @($required | Where-Object { -not (Test-Path $_) })
  if ($missing.Count -gt 0) {
    $message = "Skipping $NativeArch package because required artifacts are missing: $($missing -join ', ')"
    if ($SkipMissingArch) {
      Write-Warning $message
      continue
    }
    throw $message
  }

  $PackageName = "shurufa233-windows-$NativeArch-$Version"
  $Stage = Join-Path $DistRoot $PackageName
  $ZipPath = Join-Path $DistRoot "$PackageName.zip"
  Remove-Item -LiteralPath $Stage -Recurse -Force -ErrorAction SilentlyContinue
  Remove-Item -LiteralPath $ZipPath -Force -ErrorAction SilentlyContinue

  Copy-RequiredFile -Source (Join-Path $NativeOut "Shurufa233Tsf.dll") -Destination (Join-Path $Stage "build\windows\$NativeArch\Shurufa233Tsf.dll")
  Copy-RequiredFile -Source (Join-Path $NativeOut "Shurufa233ProfileCtl.exe") -Destination (Join-Path $Stage "build\windows\$NativeArch\Shurufa233ProfileCtl.exe")
  Copy-RequiredFile -Source (Join-Path $NativeOut "Shurufa233SmokeEdit.exe") -Destination (Join-Path $Stage "build\windows\$NativeArch\Shurufa233SmokeEdit.exe")
  Copy-RequiredFile -Source (Join-Path $GoOut "shurufa-daemon.exe") -Destination (Join-Path $Stage "build\windows\go-$GoArch\shurufa-daemon.exe")
  Copy-RequiredFile -Source (Join-Path $GoOut "shurufa-imecli.exe") -Destination (Join-Path $Stage "build\windows\go-$GoArch\shurufa-imecli.exe")
  Copy-RequiredFile -Source (Join-Path $GoOut "shurufa-dictimport.exe") -Destination (Join-Path $Stage "build\windows\go-$GoArch\shurufa-dictimport.exe")
  Copy-RequiredFile -Source (Join-Path $GoOut "shurufa-dictmanifest.exe") -Destination (Join-Path $Stage "build\windows\go-$GoArch\shurufa-dictmanifest.exe")
  Copy-RequiredFile -Source (Join-Path $GoOut "shurufa-dictsync.exe") -Destination (Join-Path $Stage "build\windows\go-$GoArch\shurufa-dictsync.exe")

  $CoreSource = Join-Path $GoOut "shurufa_core.dll"
  if (Test-Path $CoreSource) {
    Copy-Item -Force $CoreSource (Join-Path $Stage "build\windows\go-$GoArch\shurufa_core.dll")
  }

  $SettingsDist = Join-Path $Root "apps\settings\dist"
  if (-not (Test-Path (Join-Path $SettingsDist "index.html"))) {
    throw "Missing settings UI build: $SettingsDist. Run scripts\build-windows.ps1 before packaging."
  }
  New-Item -ItemType Directory -Force (Join-Path $Stage "apps\settings\dist") | Out-Null
  Copy-Item -Force -Recurse (Join-Path $SettingsDist "*") (Join-Path $Stage "apps\settings\dist")

  Copy-RequiredFile -Source (Join-Path $Root "scripts\install-windows.ps1") -Destination (Join-Path $Stage "scripts\install-windows.ps1")
  Copy-RequiredFile -Source (Join-Path $Root "scripts\uninstall-windows.ps1") -Destination (Join-Path $Stage "scripts\uninstall-windows.ps1")
  Copy-RequiredFile -Source (Join-Path $Root "docs\windows.md") -Destination (Join-Path $Stage "docs\windows.md")
  Copy-RequiredFile -Source (Join-Path $Root "docs\abi.md") -Destination (Join-Path $Stage "docs\abi.md")
  Copy-RequiredFile -Source (Join-Path $Root "docs\ipc.md") -Destination (Join-Path $Stage "docs\ipc.md")
  Copy-RequiredFile -Source (Join-Path $Root "docs\dictionaries.md") -Destination (Join-Path $Stage "docs\dictionaries.md")
  $DictionarySource = Join-Path $Root "data\dictionaries"
  if (Test-Path $DictionarySource) {
    New-Item -ItemType Directory -Force (Join-Path $Stage "data\dictionaries") | Out-Null
    Copy-Item -Force (Join-Path $DictionarySource "*.json") (Join-Path $Stage "data\dictionaries")
    Copy-Item -Force (Join-Path $DictionarySource "*.json.gz") (Join-Path $Stage "data\dictionaries") -ErrorAction SilentlyContinue
  }

  $artifacts = [System.Collections.Generic.List[object]]::new()
  $stagePrefix = (Resolve-Path -LiteralPath $Stage).Path.TrimEnd("\") + "\"
  Get-ChildItem $Stage -Recurse -File | ForEach-Object {
    $fullName = $_.FullName
    $relative = $fullName.Substring($stagePrefix.Length)
    $role = Get-ArtifactRole -RelativePath $relative -NativeArch $NativeArch -GoArch $GoArch
    Add-ArtifactHash -Artifacts $artifacts -Path $fullName -RelativePath $relative -Role $role -Required $true
  }

  $manifest = [pscustomobject]@{
    name = "shurufa233"
    version = $Version
    gitCommit = $GitCommit
    platform = "windows"
    nativeArch = $NativeArch
    goArch = $GoArch
    createdAt = (Get-Date).ToUniversalTime().ToString("o")
    install = "powershell -ExecutionPolicy Bypass -File .\scripts\install-windows.ps1 -SkipBuild"
    uninstall = "powershell -ExecutionPolicy Bypass -File .\scripts\uninstall-windows.ps1"
    coreDllPresent = (Test-Path $CoreSource)
    performanceMode = if (Test-Path $CoreSource) { "in-process-core" } else { "daemon-ipc-fallback" }
    artifacts = $artifacts
  }
  $manifestPath = Join-Path $Stage "manifest.json"
  $manifest | ConvertTo-Json -Depth 8 | Set-Content -Encoding UTF8 $manifestPath

  Compress-Archive -Path (Join-Path $Stage "*") -DestinationPath $ZipPath -Force
  Write-Host "Packaged $ZipPath"
}
