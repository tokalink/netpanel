# Create build directories
if (!(Test-Path "build")) {
    New-Item -ItemType Directory -Force -Path "build" | Out-Null
}

Write-Host "Starting cross-platform build..." -ForegroundColor Cyan

# Windows
Write-Host "Building for Windows (amd64)..."
$Env:GOOS = "windows"; $Env:GOARCH = "amd64"
go build -ldflags="-s -w" -o build/vps-panel-windows-amd64.exe ./cmd/server

# Linux
Write-Host "Building for Linux (amd64)..."
$Env:GOOS = "linux"; $Env:GOARCH = "amd64"
go build -ldflags="-s -w" -o build/vps-panel-linux-amd64 ./cmd/server

Write-Host "Building for Linux (arm64)..."
$Env:GOOS = "linux"; $Env:GOARCH = "arm64"
go build -ldflags="-s -w" -o build/vps-panel-linux-arm64 ./cmd/server

# macOS
Write-Host "Building for macOS (Intel)..."
$Env:GOOS = "darwin"; $Env:GOARCH = "amd64"
go build -ldflags="-s -w" -o build/vps-panel-darwin-amd64 ./cmd/server

Write-Host "Building for macOS (Apple Silicon)..."
$Env:GOOS = "darwin"; $Env:GOARCH = "arm64"
go build -ldflags="-s -w" -o build/vps-panel-darwin-arm64 ./cmd/server

# Cleanup env
$Env:GOOS = $null
$Env:GOARCH = $null

# Copy resources
Write-Host "Copying resources..." -ForegroundColor Cyan
Copy-Item -Path "web" -Destination "build" -Recurse -Force
Copy-Item -Path "config.yaml" -Destination "build/config.yaml" -Force

if (!(Test-Path "build/data")) {
    New-Item -ItemType Directory -Force -Path "build/data" | Out-Null
}

Write-Host "Build complete! All files are in the build directory." -ForegroundColor Green
Get-ChildItem build | Select-Object Name, Length
