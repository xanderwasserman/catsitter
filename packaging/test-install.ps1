# Exercise install.ps1: install, update, skip, and uninstall.
#Requires -Version 5.1
Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
$scriptPath = Join-Path $root 'install.ps1'
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw 'go is required'
}

$native = $env:PROCESSOR_ARCHITEW6432
if ([string]::IsNullOrEmpty($native)) {
    $native = $env:PROCESSOR_ARCHITECTURE
}
switch ($native) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { throw "unsupported architecture: $native" }
}

$work = Join-Path ([System.IO.Path]::GetTempPath()) ("catsitter-test-" + [guid]::NewGuid().ToString('n'))
$fixtureHome = Join-Path $work 'home'
$serverRoot = Join-Path $work 'www'
$marker = Join-Path $work 'marker'
New-Item -ItemType Directory -Force -Path $fixtureHome, $serverRoot | Out-Null

$server = $null
function Stop-Fixture {
    if ($script:server -and -not $script:server.HasExited) {
        Stop-Process -Id $script:server.Id -Force -ErrorAction SilentlyContinue
    }
    Get-Process -Name 'catsitter' -ErrorAction SilentlyContinue | ForEach-Object {
        try {
            $path = $_.Path
        }
        catch {
            $path = ''
        }
        if ($path -and $path.StartsWith($script:fixtureHome)) {
            Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
        }
    }
    if (Test-Path -LiteralPath $script:work) {
        Remove-Item -LiteralPath $script:work -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Publish-Release {
    param(
        [Parameter(Mandatory = $true)][string]$Version,
        [Parameter(Mandatory = $true)][string]$Binary
    )
    $dir = Join-Path $script:serverRoot "download\v$Version"
    $stage = Join-Path $script:work "stage-$Version"
    New-Item -ItemType Directory -Force -Path $dir, $stage | Out-Null
    $stagedExe = Join-Path $stage 'catsitter.exe'
    Copy-Item -LiteralPath $Binary -Destination $stagedExe -Force
    $asset = "catsitter_${Version}_windows_$($script:arch).zip"
    $zip = Join-Path $dir $asset
    if (Test-Path -LiteralPath $zip) {
        Remove-Item -LiteralPath $zip -Force
    }
    Compress-Archive -Path $stagedExe -DestinationPath $zip
    $hash = (Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash.ToLowerInvariant()
    Set-Content -LiteralPath (Join-Path $dir 'checksums.txt') -Value "$hash  $asset" -Encoding ascii
    Remove-Item -LiteralPath $stage -Recurse -Force
}

try {
    $stub = Join-Path $root 'packaging\stub'
    $one = Join-Path $work 'catsitter-0.1.0.exe'
    $two = Join-Path $work 'catsitter-0.2.0.exe'
    & go build -ldflags '-X main.version=0.1.0' -o $one $stub
    if ($LASTEXITCODE -ne 0) { throw 'stub 0.1.0 build failed' }
    & go build -ldflags '-X main.version=0.2.0' -o $two $stub
    if ($LASTEXITCODE -ne 0) { throw 'stub 0.2.0 build failed' }
    Publish-Release -Version '0.1.0' -Binary $one
    Publish-Release -Version '0.2.0' -Binary $two

    $python = Get-Command python -ErrorAction SilentlyContinue
    if (-not $python) {
        $python = Get-Command python3 -ErrorAction SilentlyContinue
    }
    if (-not $python) {
        throw 'python is required'
    }
    $listener = New-Object System.Net.Sockets.TcpListener([System.Net.IPAddress]::Loopback, 0)
    $listener.Start()
    $port = ($listener.LocalEndpoint).Port
    $listener.Stop()
    $server = Start-Process -FilePath $python.Source -ArgumentList @('-m', 'http.server', "$port", '--bind', '127.0.0.1', '--directory', $serverRoot) -PassThru -WindowStyle Hidden
    $base = "http://127.0.0.1:$port"
    Set-Content -LiteralPath (Join-Path $serverRoot 'latest') -Value "v0.0.0`n" -Encoding ascii
    $ready = $false
    foreach ($try in 1..50) {
        try {
            Invoke-WebRequest -Uri "$base/latest" -UseBasicParsing | Out-Null
            $ready = $true
            break
        }
        catch {
            Start-Sleep -Milliseconds 100
        }
    }
    if (-not $ready) {
        throw 'fixture server did not start'
    }

    $env:LOCALAPPDATA = Join-Path $fixtureHome 'Local'
    $env:APPDATA = Join-Path $fixtureHome 'Roaming'
    New-Item -ItemType Directory -Force -Path $env:LOCALAPPDATA, $env:APPDATA | Out-Null
    $env:CATSITTER_RELEASE_BASE = $base
    $env:CATSITTER_STUB_MARKER = $marker
    Remove-Item Env:CATSITTER_REPO -ErrorAction SilentlyContinue

    $installDir = Join-Path $env:LOCALAPPDATA 'Programs\Catsitter'
    $exe = Join-Path $installDir 'catsitter.exe'
    $shortcut = Join-Path $env:APPDATA 'Microsoft\Windows\Start Menu\Programs\Catsitter.lnk'

    $checksum = Join-Path $serverRoot 'download\v0.1.0\checksums.txt'
    $saved = Get-Content -LiteralPath $checksum -Raw
    $assetName = "catsitter_0.1.0_windows_$arch.zip"
    Set-Content -LiteralPath $checksum -Value ("0000000000000000000000000000000000000000000000000000000000000000  $assetName") -Encoding ascii
    Set-Content -LiteralPath (Join-Path $serverRoot 'latest') -Value "v0.1.0`n" -Encoding ascii
    $failed = $false
    try {
        & $scriptPath
    }
    catch {
        $failed = $true
    }
    if (-not $failed) {
        throw 'bad checksum was accepted'
    }
    if (Test-Path -LiteralPath $exe) {
        throw 'bad checksum still installed a binary'
    }
    Set-Content -LiteralPath $checksum -Value $saved -Encoding ascii -NoNewline

    & $scriptPath
    if (-not (Test-Path -LiteralPath $exe)) {
        throw 'binary was not installed'
    }
    $got = ((& $exe --version) | Out-String).Trim()
    if ($got -ne '0.1.0') {
        throw "installed version is $got"
    }
    if (-not (Test-Path -LiteralPath $shortcut)) {
        throw 'missing Start Menu shortcut'
    }
    $shell = New-Object -ComObject WScript.Shell
    $link = $shell.CreateShortcut($shortcut)
    if ($link.TargetPath -ne $exe) {
        throw "shortcut target is $($link.TargetPath)"
    }

    Set-Content -LiteralPath (Join-Path $serverRoot 'latest') -Value "v0.2.0`n" -Encoding ascii
    & $scriptPath -Version '0.1.0'
    $got = ((& $exe --version) | Out-String).Trim()
    if ($got -ne '0.1.0') {
        throw "pinned install changed the version to $got"
    }

    if (Test-Path -LiteralPath $marker) {
        Remove-Item -LiteralPath $marker -Force
    }
    Start-Process -FilePath $exe -WorkingDirectory $installDir | Out-Null
    $ready = $false
    foreach ($try in 1..50) {
        if ((Test-Path -LiteralPath $marker) -and ((Get-Item $marker).Length -gt 0)) {
            $ready = $true
            break
        }
        Start-Sleep -Milliseconds 100
    }
    if (-not $ready) {
        throw 'stub did not write a marker'
    }
    $before = (Get-Content -LiteralPath $marker -Raw).Trim()

    & $scriptPath
    $got = ((& $exe --version) | Out-String).Trim()
    if ($got -ne '0.2.0') {
        throw "update left version $got"
    }
    $after = $before
    foreach ($try in 1..50) {
        $after = (Get-Content -LiteralPath $marker -Raw).Trim()
        if ($after -and $after -ne $before) {
            break
        }
        Start-Sleep -Milliseconds 100
    }
    if ($after -eq $before) {
        throw 'update did not restart Catsitter'
    }

    $sumBefore = (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash
    & $scriptPath
    $sumAfter = (Get-FileHash -LiteralPath $exe -Algorithm SHA256).Hash
    if ($sumBefore -ne $sumAfter) {
        throw 'skip replaced the current binary'
    }
    $still = (Get-Content -LiteralPath $marker -Raw).Trim()
    if ($still -ne $after) {
        throw 'skip stopped the running Catsitter'
    }

    & $scriptPath -Uninstall
    if (Test-Path -LiteralPath $exe) {
        throw 'uninstall left the binary'
    }
    if (Test-Path -LiteralPath $shortcut) {
        throw 'uninstall left the shortcut'
    }
    $gone = $false
    foreach ($try in 1..50) {
        $proc = Get-Process -Id $after -ErrorAction SilentlyContinue
        if (-not $proc) {
            $gone = $true
            break
        }
        Start-Sleep -Milliseconds 100
    }
    if (-not $gone) {
        throw 'uninstall left Catsitter running'
    }

    # The double-dash flag has to uninstall a fresh copy too.
    Set-Content -LiteralPath (Join-Path $serverRoot 'latest') -Value "v0.1.0`n" -Encoding ascii
    & $scriptPath --version '0.1.0'
    $got = ((& $exe --version) | Out-String).Trim()
    if ($got -ne '0.1.0') {
        throw "dash-dash version install returned $got"
    }
    & $scriptPath --uninstall
    if (Test-Path -LiteralPath $exe) {
        throw 'dash-dash uninstall left the binary'
    }

    Write-Output "install, update, and uninstall passed (windows/$arch)"
}
finally {
    Stop-Fixture
}
