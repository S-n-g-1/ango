<#
.SYNOPSIS
  Builds Ango on Windows without needing make.
.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts\build.ps1
  powershell -ExecutionPolicy Bypass -File scripts\build.ps1 -Version 0.1.1 -Dist
#>
param(
    [string]$Version = "dev",
    [switch]$Dist
)
$ErrorActionPreference = "Stop"
$env:CGO_ENABLED = "0"
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
Set-Location $root

New-Item -ItemType Directory -Force bin | Out-Null
go build -trimpath -ldflags "-s -w -X main.version=$Version" -o bin\ango.exe .\cmd\ango
go build -trimpath -ldflags "-s -w -H=windowsgui" -o bin\ango-launcher.exe .\cmd\ango-launcher
Write-Host "Built bin\ango.exe and bin\ango-launcher.exe"

if ($Dist) {
    $arch = go env GOARCH
    $name = "ango-$Version-windows-$arch"
    $pkg = Join-Path "dist" $name
    Remove-Item -Recurse -Force $pkg -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force (Join-Path $pkg "projects") | Out-Null
    Copy-Item bin\ango.exe, bin\ango-launcher.exe $pkg
    Copy-Item scripts\ango.bat, scripts\play.bat $pkg
    Copy-Item -Recurse examples\intro (Join-Path $pkg "projects\intro")
    Copy-Item README.md, LICENSE $pkg
    Copy-Item -Recurse docs (Join-Path $pkg "docs")
    Compress-Archive -Force -Path $pkg -DestinationPath "dist\$name.zip"
    Write-Host "Created dist\$name.zip"
}
