param(
  [switch]$UnregisterOnly,
  [string]$TsfDllPath,
  [switch]$RemoveUserData
)

$ErrorActionPreference = "Stop"

$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\shurufa233"
$NativeInstallDir = Join-Path $env:ProgramFiles "shurufa233"
$ConfigDir = Join-Path $env:APPDATA "shurufa233"
$InputMethodBackupPath = Join-Path $ConfigDir "input-method-backup.json"
$RunKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run"
$RunName = "shurufa233-daemon"
$Tip = "0804:{3D7B8D06-9872-4C31-B77D-3B87327CBF64}{B68911A2-4478-491C-A624-978441648E20}"
$ClsidKey = "HKLM:\SOFTWARE\Classes\CLSID\{3D7B8D06-9872-4C31-B77D-3B87327CBF64}\InProcServer32"

function Test-IsAdmin {
  $principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
  return $principal.IsInRole([Security.Principal.WindowsBuiltinRole]::Administrator)
}

function Get-RegistryTsfDll {
  try {
    return (Get-ItemProperty $ClsidKey -ErrorAction Stop)."(default)"
  } catch {
    return $null
  }
}

function Get-TsfDllForUnregister {
  if ($TsfDllPath) {
    return $TsfDllPath
  }
  return Get-RegistryTsfDll
}

function Invoke-ElevatedUnregister {
  $registered = Get-TsfDllForUnregister
  $args = @(
    "-NoProfile",
    "-ExecutionPolicy", "Bypass",
    "-File", "`"$PSCommandPath`"",
    "-UnregisterOnly"
  )
  if ($registered) {
    $args += @("-TsfDllPath", "`"$registered`"")
  }
  $proc = Start-Process -FilePath "powershell.exe" -ArgumentList $args -Verb RunAs -Wait -PassThru
  if ($proc.ExitCode -ne 0) {
    throw "Elevated TSF unregister failed with exit code $($proc.ExitCode)."
  }

  $remaining = Get-RegistryTsfDll
  if ($remaining -and $registered -and ($remaining -eq $registered)) {
    throw "TSF unregister did not remove HKLM COM registration for $registered"
  }
}

function Unregister-Tsf {
  $registered = Get-TsfDllForUnregister
  if ($registered -and (Test-Path $registered)) {
    regsvr32.exe /u /s $registered
  }

  $remaining = Get-RegistryTsfDll
  if ($remaining -and $remaining -eq $registered) {
    throw "TSF unregister did not remove HKLM COM registration for $registered"
  }
}

function ConvertTo-BackupLanguages {
  param($Backup)

  if ($Backup.Languages) {
    return @($Backup.Languages)
  }
  if ($Backup.LanguageTag) {
    return @([pscustomobject]@{
      LanguageTag = $Backup.LanguageTag
      InputMethodTips = @($Backup.InputMethodTips)
    })
  }
  return @()
}

function Restore-InputMethods {
  if (Test-Path $InputMethodBackupPath) {
    $backup = Get-Content $InputMethodBackupPath -Raw | ConvertFrom-Json
    $backupLanguages = ConvertTo-BackupLanguages $backup
    if ($backupLanguages.Count -gt 0) {
      $restored = $null
      foreach ($item in $backupLanguages) {
        if (-not $item.LanguageTag) { continue }
        $language = New-WinUserLanguageList $item.LanguageTag
        $entry = $language[0]
        $tips = @($item.InputMethodTips | Where-Object { $_ -and $_ -ne $Tip })
        if ($tips.Count -gt 0) {
          $entry.InputMethodTips.Clear()
        }
        foreach ($tipValue in $tips) {
          if ($tipValue) {
            $entry.InputMethodTips.Add([string]$tipValue)
          }
        }
        if ($null -eq $restored) {
          $restored = $language
        } else {
          $restored.Add($entry)
        }
      }
      if ($restored -and $restored.Count -gt 0) {
        Set-WinUserLanguageList $restored -Force
      }
    }

    if ($backup.DefaultInputMethodTip -and $backup.DefaultInputMethodTip -ne $Tip) {
      Set-WinDefaultInputMethodOverride -InputTip $backup.DefaultInputMethodTip
    } else {
      Set-WinDefaultInputMethodOverride
    }
    Write-Host "Restored input methods from $InputMethodBackupPath"
    return
  }

  $languages = Get-WinUserLanguageList
  $changed = $false
  foreach ($language in $languages) {
    if ($language.InputMethodTips -contains $Tip) {
      $language.InputMethodTips.Remove($Tip)
      $changed = $true
    }
  }
  if ($changed) {
    Set-WinUserLanguageList $languages -Force
  }

  $defaultOverride = Get-WinDefaultInputMethodOverride -ErrorAction SilentlyContinue
  if ($defaultOverride.InputMethodTip -eq $Tip) {
    Set-WinDefaultInputMethodOverride
  }
}

function Remove-DirectoryTree {
  param(
    [string]$Path,
    [string]$ExpectedParent
  )

  if (-not (Test-Path $Path)) {
    return
  }

  $resolvedPath = (Resolve-Path -LiteralPath $Path).Path
  $resolvedParent = (Resolve-Path -LiteralPath $ExpectedParent).Path
  $expectedPrefix = $resolvedParent.TrimEnd("\") + "\"
  if (-not $resolvedPath.StartsWith($expectedPrefix, [StringComparison]::OrdinalIgnoreCase)) {
    throw "Refusing to remove unexpected path: $resolvedPath"
  }
  if ((Split-Path $resolvedPath -Leaf) -ne "shurufa233") {
    throw "Refusing to remove unexpected directory name: $resolvedPath"
  }

  Remove-Item -LiteralPath $resolvedPath -Recurse -Force -ErrorAction SilentlyContinue
}

if ($UnregisterOnly) {
  if (-not (Test-IsAdmin)) {
    throw "UnregisterOnly requires an elevated PowerShell session."
  }
  Unregister-Tsf
  Write-Host "Unregistered shurufa233 TSF profile."
  exit 0
}

Remove-ItemProperty -Path $RunKey -Name $RunName -ErrorAction SilentlyContinue
Get-Process -Name shurufa-daemon -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
for ($i = 0; $i -lt 20; $i++) {
  if (-not (Get-Process -Name shurufa-daemon -ErrorAction SilentlyContinue)) { break }
  Start-Sleep -Milliseconds 250
}

if (Test-IsAdmin) {
  Unregister-Tsf
} else {
  Invoke-ElevatedUnregister
}

Restore-InputMethods

Remove-DirectoryTree -Path $InstallDir -ExpectedParent (Join-Path $env:LOCALAPPDATA "Programs")
Remove-DirectoryTree -Path $NativeInstallDir -ExpectedParent $env:ProgramFiles
if ($RemoveUserData) {
  Remove-DirectoryTree -Path $ConfigDir -ExpectedParent $env:APPDATA
}

Start-Process ctfmon.exe -WindowStyle Hidden -ErrorAction SilentlyContinue

Write-Host "Uninstalled shurufa233."
if (-not $RemoveUserData) {
  Write-Host "User data was kept at $ConfigDir"
}
