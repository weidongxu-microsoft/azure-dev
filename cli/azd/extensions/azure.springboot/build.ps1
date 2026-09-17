$ErrorActionPreference = 'Stop'

$extensionDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location -Path $extensionDir

$extensionId = $env:EXTENSION_ID -replace '\.', '-'
$outputDir = if ($env:OUTPUT_DIR) { $env:OUTPUT_DIR } else { Join-Path $extensionDir 'bin' }
New-Item -ItemType Directory -Path $outputDir -Force | Out-Null

$platforms = if ($env:EXTENSION_PLATFORM) {
    @($env:EXTENSION_PLATFORM)
}
else {
    @(
        'windows/amd64',
        'windows/arm64',
        'darwin/amd64',
        'darwin/arm64',
        'linux/amd64',
        'linux/arm64'
    )
}

foreach ($platform in $platforms) {
    $os, $arch = $platform -split '/'
    $outputName = Join-Path $outputDir "$extensionId-$os-$arch"
    if ($os -eq 'windows') {
        $outputName += '.exe'
    }

    $env:GOOS = $os
    $env:GOARCH = $arch
    go build -o $outputName
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
}
