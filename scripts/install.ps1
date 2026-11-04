<#
.SYNOPSIS
  Build zipstar for this machine and install it, plus a matching UnRAR binary.

.DESCRIPTION
  Installs per-user (no admin needed) to %LOCALAPPDATA%\Programs\zipstar by
  default and adds that folder to the user PATH.

.EXAMPLE
  .\scripts\install.ps1
  .\scripts\install.ps1 -Prefix C:\tools\zipstar
  .\scripts\install.ps1 -NoUnrar
  .\scripts\install.ps1 -Uninstall
#>
[CmdletBinding()]
param(
  [string]$Prefix = (Join-Path $env:LOCALAPPDATA 'Programs\zipstar'),
  [string]$Unrar = '',   # install this UnRAR.exe instead of fetching one
  [switch]$NoUnrar,      # skip UnRAR; zipstar falls back to its built-in RAR decoder
  [switch]$NoPath,       # never edit the user PATH
  [switch]$Uninstall
)
$ErrorActionPreference = 'Stop'

$Root = Split-Path -Parent $PSScriptRoot
$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }
# RARLab publishes an x64 UnRAR only; it runs under emulation on Windows ARM64.
$UnrarName = 'unrar-windows-amd64.exe'
$Prefix = [IO.Path]::GetFullPath($Prefix).TrimEnd('\')
$ZipstarExe = Join-Path $Prefix 'zipstar.exe'

# Read/write the raw user PATH so %VAR% entries keep their REG_EXPAND_SZ form.
function Get-UserPath {
  $key = Get-Item 'HKCU:\Environment'
  return [string]$key.GetValue('Path', '', 'DoNotExpandEnvironmentNames')
}
function Set-UserPath([string]$Value) {
  [Microsoft.Win32.Registry]::SetValue('HKEY_CURRENT_USER\Environment', 'Path', $Value,
    [Microsoft.Win32.RegistryValueKind]::ExpandString)
  # Setting any user variable broadcasts WM_SETTINGCHANGE so new terminals see the change.
  [Environment]::SetEnvironmentVariable('ZIPSTAR_PATH_REFRESH', '1', 'User')
  [Environment]::SetEnvironmentVariable('ZIPSTAR_PATH_REFRESH', $null, 'User')
}
function Split-PathList([string]$Value) {
  return @($Value -split ';' | Where-Object { $_ })
}
function Test-SameDir([string]$A, [string]$B) {
  return $A.TrimEnd('\') -ieq $B.TrimEnd('\')
}

if ($Uninstall) {
  foreach ($f in @($ZipstarExe, (Join-Path $Prefix $UnrarName), (Join-Path $Prefix 'UNRAR-LICENSE.txt'))) {
    Remove-Item -Force -ErrorAction SilentlyContinue $f
  }
  if ((Test-Path $Prefix) -and -not (Get-ChildItem $Prefix)) { Remove-Item $Prefix }
  $kept = Split-PathList (Get-UserPath) | Where-Object { -not (Test-SameDir $_ $Prefix) }
  Set-UserPath ($kept -join ';')
  Write-Host "uninstalled zipstar from $Prefix"
  return
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
  throw 'Go is required to build zipstar (https://go.dev/dl/)'
}

$tmp = Join-Path ([IO.Path]::GetTempPath()) ('zipstar-install-' + [guid]::NewGuid())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
  $Version = 'dev'
  if (Get-Command git -ErrorAction SilentlyContinue) {
    $described = & git -C $Root describe --tags --always --dirty 2>$null
    if ($LASTEXITCODE -eq 0 -and $described) { $Version = $described.Trim() }
  }
  Write-Host "building zipstar $Version for windows_$Arch..."
  $builtExe = Join-Path $tmp 'zipstar.exe'
  Push-Location $Root
  try {
    & go build -trimpath -ldflags "-s -w -X github.com/fr13dman/zipstar-cli/internal/version.Version=$Version" -o $builtExe ./cmd/zipstar
    if ($LASTEXITCODE -ne 0) { throw 'go build failed' }
  } finally { Pop-Location }

  $unrarBin = $null
  if (-not $NoUnrar) {
    $local = Join-Path $Root 'third_party\unrar\windows_amd64\unrar.exe'
    if ($Unrar) {
      if (-not (Test-Path $Unrar)) { throw "-Unrar file not found: $Unrar" }
      $unrarBin = $Unrar
    } elseif (Test-Path $local) {
      $unrarBin = $local
    } else {
      try {
        Write-Host 'fetching UnRAR from rarlab.com...'
        # unrarw64.exe is a self-extracting RAR (UnRAR.exe + license.txt). Saved
        # with a .rar name, zipstar's built-in decoder can unpack it.
        $sfx = Join-Path $tmp 'unrar-sfx.rar'
        Invoke-WebRequest -UseBasicParsing -Uri 'https://www.rarlab.com/rar/unrarw64.exe' -OutFile $sfx
        $env:ZIPSTAR_UNRAR = 'off'
        & $builtExe u -P=false -F -C (Join-Path $tmp 'unrar') $sfx | Out-Null
        if ($LASTEXITCODE -ne 0) { throw 'extracting UnRAR failed' }
        $unrarBin = Join-Path $tmp 'unrar\UnRAR.exe'
        if (-not (Test-Path $unrarBin)) { throw 'UnRAR.exe missing from download' }
      } catch {
        Write-Warning "UnRAR not installed ($_); zipstar will use its built-in RAR decoder"
        $unrarBin = $null
      } finally {
        Remove-Item Env:ZIPSTAR_UNRAR -ErrorAction SilentlyContinue
      }
    }
  }

  New-Item -ItemType Directory -Force -Path $Prefix | Out-Null
  Copy-Item -Force $builtExe $ZipstarExe
  if ($unrarBin) {
    Copy-Item -Force $unrarBin (Join-Path $Prefix $UnrarName)
    Copy-Item -Force (Join-Path $Root 'third_party\unrar\LICENSE.txt') (Join-Path $Prefix 'UNRAR-LICENSE.txt')
  }

  & $ZipstarExe --help | Out-Null
  if ($LASTEXITCODE -ne 0) { throw 'installed zipstar failed to run' }
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

Write-Host "installed zipstar $Version -> $ZipstarExe"
if ($unrarBin) { Write-Host "installed UnRAR   -> $(Join-Path $Prefix $UnrarName)" }

$onPath = (Split-PathList $env:Path) + (Split-PathList (Get-UserPath)) | Where-Object { Test-SameDir $_ $Prefix }
if ($onPath) {
  Write-Host "$Prefix is already on PATH"
} elseif ($NoPath) {
  Write-Host "$Prefix is not on PATH; add it to run zipstar from anywhere"
} else {
  Set-UserPath (((Split-PathList (Get-UserPath)) + $Prefix) -join ';')
  $env:Path = "$env:Path;$Prefix"
  Write-Host "added $Prefix to your user PATH (open a new terminal to use zipstar)"
}
