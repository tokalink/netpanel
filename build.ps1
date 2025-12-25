# Create build directory
if (!(Test-Path "build")) {
    New-Item -ItemType Directory -Force -Path "build" | Out-Null
}

Write-Host "Starting cross-platform build..." -ForegroundColor Cyan

# Windows (amd64)
Write-Host "Building for Windows (amd64)..."
$Env:GOOS = "windows"; $Env:GOARCH = "amd64"
go build -ldflags="-s -w" -o build/vps-panel-windows-amd64.exe ./cmd/server

# Linux (amd64)
Write-Host "Building for Linux (amd64)..."
$Env:GOOS = "linux"; $Env:GOARCH = "amd64"
go build -ldflags="-s -w" -o build/vps-panel-linux-amd64 ./cmd/server

# Linux (arm64)
Write-Host "Building for Linux (arm64)..."
$Env:GOOS = "linux"; $Env:GOARCH = "arm64"
go build -ldflags="-s -w" -o build/vps-panel-linux-arm64 ./cmd/server

# macOS (Intel)
Write-Host "Building for macOS (Intel)..."
$Env:GOOS = "darwin"; $Env:GOARCH = "amd64"
go build -ldflags="-s -w" -o build/vps-panel-darwin-amd64 ./cmd/server

# macOS (Apple Silicon)
Write-Host "Building for macOS (Apple Silicon)..."
$Env:GOOS = "darwin"; $Env:GOARCH = "arm64"
go build -ldflags="-s -w" -o build/vps-panel-darwin-arm64 ./cmd/server

# Clean up environment variables
$Env:GOOS = $null
$Env:GOARCH = $null

Write-Host "Build complete! Binaries are in the 'build' directory." -ForegroundColor Green
Get-ChildItem build | Select-Object Name, Length
