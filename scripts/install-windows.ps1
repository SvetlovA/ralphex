[CmdletBinding()]
param(
    [string]$InstallDirectory = "$env:LOCALAPPDATA\Programs\ralphex"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$repository = "SvetlovA/ralphex"
$executable = "ralphex.windows.exe"
$architecture = switch ([System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    "X64" { "amd64" }
    "Arm64" { "arm64" }
    default { throw "Unsupported Windows architecture: $_" }
}

$headers = @{ "User-Agent" = "ralphex-windows-installer" }
$release = Invoke-RestMethod -Headers $headers -Uri "https://api.github.com/repos/$repository/releases/latest"
$assetPattern = "^ralphex_windows_.*_${architecture}\.zip$"
$asset = @($release.assets | Where-Object { $_.name -match $assetPattern })

if ($asset.Count -ne 1) {
    throw "Expected one $architecture Windows archive in release $($release.tag_name), found $($asset.Count)."
}

$temporaryDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("ralphex-" + [guid]::NewGuid())
$archivePath = Join-Path $temporaryDirectory $asset[0].name
$extractPath = Join-Path $temporaryDirectory "extract"

try {
    New-Item -ItemType Directory -Path $extractPath -Force | Out-Null
    Invoke-WebRequest -Headers $headers -Uri $asset[0].browser_download_url -OutFile $archivePath
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractPath -Force

    $sourceExecutable = Join-Path $extractPath $executable
    if (-not (Test-Path -LiteralPath $sourceExecutable -PathType Leaf)) {
        throw "The release archive does not contain $executable."
    }

    New-Item -ItemType Directory -Path $InstallDirectory -Force | Out-Null
    Copy-Item -LiteralPath $sourceExecutable -Destination (Join-Path $InstallDirectory $executable) -Force

    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @($userPath -split ";" | Where-Object { $_ })
    if ($pathEntries -notcontains $InstallDirectory) {
        $newUserPath = (@($pathEntries) + $InstallDirectory) -join ";"
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        $env:Path = "$env:Path;$InstallDirectory"
    }

    Write-Host "Installed $executable from $($release.tag_name) to $InstallDirectory"
    Write-Host "Run it as: ralphex.windows.exe --version"
}
finally {
    if (Test-Path -LiteralPath $temporaryDirectory) {
        Remove-Item -LiteralPath $temporaryDirectory -Recurse -Force
    }
}
