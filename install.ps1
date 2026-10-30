# Install Catsitter for the current user. No administrator rights.
#
#   irm https://github.com/xanderwasserman/catsitter/releases/latest/download/install.ps1 | iex
#   iex "& { $(irm https://github.com/xanderwasserman/catsitter/releases/latest/download/install.ps1) } -Version 0.1.0"
#   iex "& { $(irm https://github.com/xanderwasserman/catsitter/releases/latest/download/install.ps1) } -Uninstall"
#   iex "& { $(irm https://github.com/xanderwasserman/catsitter/releases/latest/download/install.ps1) } --uninstall"
#Requires -Version 5.1

[CmdletBinding()]
param(
    [string]$Version = '',
    [Alias('uninstall')]
    [switch]$Uninstall,
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$Rest
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

# PowerShell 7 binds --uninstall to -Uninstall. Older hosts pass the flag
# through as text; accept that spelling too.
if ($Rest) {
    $index = 0
    while ($index -lt $Rest.Count) {
        $item = $Rest[$index]
        if ($item -eq '--uninstall') {
            $Uninstall = $true
        }
        elseif ($item -eq '--version') {
            if (($index + 1) -ge $Rest.Count) {
                throw 'catsitter install: --version needs a value'
            }
            $Version = $Rest[$index + 1]
            $index++
        }
        elseif ($item -like '--version=*') {
            $Version = $item.Substring(10)
        }
        else {
            throw "catsitter install: unknown argument: $item"
        }
        $index++
    }
}

function Write-InstallMessage {
    param([Parameter(Mandatory = $true)][string]$Message)
    Write-Output $Message
}

function Write-InstallFailure {
    param([Parameter(Mandatory = $true)][string]$Message)
    throw "catsitter install: $Message"
}

function Get-CatsitterArch {
    $native = $env:PROCESSOR_ARCHITEW6432
    if ([string]::IsNullOrEmpty($native)) {
        $native = $env:PROCESSOR_ARCHITECTURE
    }
    switch ($native) {
        'AMD64' { return 'amd64' }
        'ARM64' { return 'arm64' }
        default { Write-InstallFailure "unsupported architecture: $native" }
    }
}

function ConvertTo-PlainVersion {
    param([AllowEmptyString()][string]$Value)
    if ([string]::IsNullOrWhiteSpace($Value)) {
        return ''
    }
    return ($Value.Trim() -replace '^v', '')
}

$repo = $env:CATSITTER_REPO
if ([string]::IsNullOrEmpty($repo)) {
    $repo = 'xanderwasserman/catsitter'
}
$releaseBase = $env:CATSITTER_RELEASE_BASE
$installDir = Join-Path $env:LOCALAPPDATA 'Programs\Catsitter'
$installExe = Join-Path $installDir 'catsitter.exe'
$shortcutPath = Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Catsitter.lnk'

function Get-InstalledVersion {
    if (-not (Test-Path -LiteralPath $installExe)) {
        return ''
    }
    try {
        $output = & $installExe --version 2>$null
        return (ConvertTo-PlainVersion ([string]$output))
    }
    catch {
        return ''
    }
}

function Test-CatsitterRunning {
    $found = @(Get-Process -Name 'catsitter' -ErrorAction SilentlyContinue)
    return ($found.Count -gt 0)
}

function Wait-CatsitterExit {
    if (-not (Test-CatsitterRunning)) {
        return
    }
    Stop-Process -Name 'catsitter' -Force -ErrorAction SilentlyContinue
    $deadline = (Get-Date).AddSeconds(10)
    while (Test-CatsitterRunning) {
        if ((Get-Date) -gt $deadline) {
            Write-InstallFailure 'Catsitter is still running'
        }
        Start-Sleep -Milliseconds 200
    }
}

function Clear-CatsitterInstall {
    if (Test-Path -LiteralPath $shortcutPath) {
        Remove-Item -LiteralPath $shortcutPath -Force
    }
    if (Test-Path -LiteralPath $installDir) {
        Remove-Item -LiteralPath $installDir -Recurse -Force
    }
}

function Invoke-CatsitterLaunch {
    Start-Process -FilePath $installExe -WorkingDirectory $installDir | Out-Null
}

if ($Uninstall) {
    Wait-CatsitterExit
    Clear-CatsitterInstall
    Write-InstallMessage 'Uninstalled Catsitter. Settings were left in place.'
    return
}

try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
}
catch {
    Write-Verbose 'TLS defaults are already current.'
}

$arch = Get-CatsitterArch
$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("catsitter-install-" + [guid]::NewGuid().ToString('n'))
New-Item -ItemType Directory -Path $tempRoot | Out-Null
try {
    $ver = ConvertTo-PlainVersion $Version
    if ([string]::IsNullOrEmpty($ver)) {
        $latestPath = Join-Path $tempRoot 'latest'
        if ([string]::IsNullOrEmpty($releaseBase)) {
            $latestUrl = "https://api.github.com/repos/$repo/releases/latest"
        }
        else {
            $latestUrl = ($releaseBase.TrimEnd('/') + '/latest')
        }
        Invoke-WebRequest -Uri $latestUrl -OutFile $latestPath -UseBasicParsing
        $latestText = Get-Content -LiteralPath $latestPath -Raw
        $tag = ''
        if ($latestText -match '"tag_name"\s*:\s*"([^"]+)"') {
            $tag = $Matches[1]
        }
        else {
            $tag = $latestText.Trim()
        }
        $ver = ConvertTo-PlainVersion $tag
    }
    if ([string]::IsNullOrEmpty($ver)) {
        Write-InstallFailure 'could not determine the version'
    }

    $current = Get-InstalledVersion
    if ($current -eq $ver) {
        Write-InstallMessage "Catsitter $ver is already installed."
        return
    }

    $tag = "v$ver"
    $asset = "catsitter_${ver}_windows_${arch}.zip"
    if ([string]::IsNullOrEmpty($releaseBase)) {
        $base = "https://github.com/$repo/releases/download/$tag"
    }
    else {
        $base = ($releaseBase.TrimEnd('/') + "/download/$tag")
    }

    Write-InstallMessage "Downloading Catsitter $ver (windows/$arch)"
    $checksumPath = Join-Path $tempRoot 'checksums.txt'
    $assetPath = Join-Path $tempRoot $asset
    Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $checksumPath -UseBasicParsing
    Invoke-WebRequest -Uri "$base/$asset" -OutFile $assetPath -UseBasicParsing

    $expected = ''
    foreach ($line in (Get-Content -LiteralPath $checksumPath)) {
        if ($line -match '^([0-9A-Fa-f]{64})\s+\*?(\S+)$') {
            if ($Matches[2] -eq $asset) {
                $expected = $Matches[1].ToLowerInvariant()
                break
            }
        }
    }
    if ([string]::IsNullOrEmpty($expected)) {
        Write-InstallFailure "checksums.txt has no entry for $asset"
    }
    $got = (Get-FileHash -LiteralPath $assetPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($got -ne $expected) {
        Write-InstallFailure "checksum mismatch for $asset"
    }

    $extract = Join-Path $tempRoot 'extract'
    Expand-Archive -LiteralPath $assetPath -DestinationPath $extract -Force
    $source = Join-Path $extract 'catsitter.exe'
    if (-not (Test-Path -LiteralPath $source)) {
        Write-InstallFailure 'archive did not contain catsitter.exe'
    }

    $wasRunning = Test-CatsitterRunning
    if ($wasRunning) {
        Wait-CatsitterExit
    }

    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    $staged = Join-Path $installDir 'catsitter.exe.new'
    Copy-Item -LiteralPath $source -Destination $staged -Force
    Move-Item -LiteralPath $staged -Destination $installExe -Force

    $menuDir = Split-Path -Parent $shortcutPath
    New-Item -ItemType Directory -Force -Path $menuDir | Out-Null
    $shell = New-Object -ComObject WScript.Shell
    $shortcut = $shell.CreateShortcut($shortcutPath)
    $shortcut.TargetPath = $installExe
    $shortcut.WorkingDirectory = $installDir
    $shortcut.Description = 'A little cat that keeps the computer awake'
    $shortcut.Save()

    Write-InstallMessage "Installed Catsitter $ver to $installDir"
    if ($wasRunning) {
        Invoke-CatsitterLaunch
        Write-InstallMessage 'Restarted Catsitter.'
    }
}
finally {
    if (Test-Path -LiteralPath $tempRoot) {
        Remove-Item -LiteralPath $tempRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
}
