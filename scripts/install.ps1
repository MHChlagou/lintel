<#
.SYNOPSIS
  Install Lintel on Windows.

.DESCRIPTION
  Downloads the lintel release binary for the current architecture,
  verifies its SHA256 against the .sha256 sidecar shipped with every
  release, optionally verifies the Sigstore bundle with cosign, and
  installs to $env:USERPROFILE\bin by default.

.PARAMETER Version
  Release tag to install (e.g. v0.2.3). Default: latest.

.PARAMETER InstallDir
  Destination directory. Default: $env:USERPROFILE\bin.

.PARAMETER NoCosign
  Skip the cosign signature verification step. SHA256 is still checked.

.EXAMPLE
  # Pipe install (non-interactive)
  iwr https://raw.githubusercontent.com/MHChlagou/lintel/main/scripts/install.ps1 -UseBasicParsing | iex

.EXAMPLE
  # Pin a specific version
  .\install.ps1 -Version v0.2.3

.NOTES
  Windows locks running executables. Close any open lintel.exe processes
  before re-running this script to upgrade in place.
#>

[CmdletBinding()]
param(
  [string]$Version = $(if ($env:LINTEL_VERSION) { $env:LINTEL_VERSION } else { 'latest' }),
  [string]$InstallDir = $(if ($env:LINTEL_INSTALL_DIR) { $env:LINTEL_INSTALL_DIR } else { Join-Path $env:USERPROFILE 'bin' }),
  [switch]$NoCosign
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'  # suppresses iwr progress spam
$repo = 'MHChlagou/lintel'

# Architecture detection — currently only amd64 ships for Windows, but
# keep the lookup table in place so adding arm64 later is one line.
$rtArch = [System.Runtime.InteropServices.RuntimeInformation]::ProcessArchitecture
$arch = switch ($rtArch) {
  'X64'   { 'amd64' }
  'Arm64' { 'arm64' }
  default { throw "lintel-install: unsupported architecture: $rtArch" }
}

$asset = "lintel-windows-$arch.exe"
$base = if ($Version -eq 'latest') {
  "https://github.com/$repo/releases/latest/download"
} else {
  "https://github.com/$repo/releases/download/$Version"
}
$url = "$base/$asset"

$tmp = New-Item -ItemType Directory -Path (Join-Path $env:TEMP ("lintel-install-" + [System.IO.Path]::GetRandomFileName()))
try {
  Write-Host "↓ downloading $url"
  Invoke-WebRequest -UseBasicParsing -Uri $url              -OutFile (Join-Path $tmp 'lintel.exe')
  Invoke-WebRequest -UseBasicParsing -Uri "$url.sha256"     -OutFile (Join-Path $tmp 'lintel.sha256')

  $expected = (Get-Content (Join-Path $tmp 'lintel.sha256') -Raw).Trim().Split()[0].ToLower()
  $actual = (Get-FileHash -Algorithm SHA256 (Join-Path $tmp 'lintel.exe')).Hash.ToLower()
  if ($expected -ne $actual) {
    throw "lintel-install: SHA256 mismatch — refusing to install.`n  expected: $expected`n  actual:   $actual"
  }
  Write-Host "✓ sha256 verified"

  # cosign is optional. In default (auto) mode we use it only when it is
  # already on PATH — pushing users to install cosign just to run this
  # script would nudge them toward -NoCosign, which defeats the point.
  $verifyCosign = -not $NoCosign -and (Get-Command cosign -ErrorAction SilentlyContinue)
  if ($verifyCosign) {
    Invoke-WebRequest -UseBasicParsing -Uri "$url.sigstore" -OutFile (Join-Path $tmp 'lintel.sigstore')
    & cosign verify-blob `
      --bundle (Join-Path $tmp 'lintel.sigstore') `
      --certificate-identity-regexp "^https://github.com/$repo/" `
      --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' `
      (Join-Path $tmp 'lintel.exe') | Out-Null
    if ($LASTEXITCODE -ne 0) {
      throw "lintel-install: cosign verification failed"
    }
    Write-Host "✓ cosign signature verified"
  } elseif (-not $NoCosign) {
    Write-Host "• cosign not installed; skipping signature verification (install cosign for stronger guarantees)"
  }

  if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
  }
  $dest = Join-Path $InstallDir 'lintel.exe'
  Move-Item -Force -Path (Join-Path $tmp 'lintel.exe') -Destination $dest
  Write-Host "✓ installed $dest"
  Write-Host ""
  & $dest version

  # PATH hint — we don't auto-mutate the user's PATH because that is a
  # surprising side effect of an install script. Show the exact command
  # instead and let the user decide.
  $userPath = [Environment]::GetEnvironmentVariable('PATH', 'User')
  if ($userPath -notlike "*$InstallDir*") {
    Write-Host ""
    Write-Host "NOTE: $InstallDir is not on your user PATH. To add it, run:"
    Write-Host "  [Environment]::SetEnvironmentVariable('PATH', `"$InstallDir;`$env:PATH`", 'User')"
    Write-Host "Then open a new shell."
  }
} finally {
  Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $tmp
}
