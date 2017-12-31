# Write-Host "Getting GitVersion"
# choco install gitversion.portable -pre -y

Write-Host "Generating version info using GitVersion"
$VersionInfo = gitversion | convertfrom-json

Write-Host ("Version detected as '{0}'" -f $VersionInfo.SemVer)

# Build binaries
## Win x86
$ENV:GOOS = "windows"; $env:GOARCH = "386"
go build -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" -o .\bin\rmstale-win_x86.exe rmstale.go

## Win AMD64
$ENV:GOOS = "windows"; $env:GOARCH = "amd64"
go build -race -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)"-o .\bin\rmstale-win_amd64.exe rmstale.go 

## Linux x86
$ENV:GOOS = "linux"; $env:GOARCH = "386"
go build -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" -o .\bin\rmstale-linux_x86 rmstale.go

## Win AMD64
$ENV:GOOS = "linux"; $env:GOARCH = "amd64"
go build -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" -o .\bin\rmstale-linux_amd64 rmstale.go

## Mac x86
$ENV:GOOS = "darwin"; $env:GOARCH = "386"
go build -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" -o .\bin\rmstale-mac_x86 rmstale.go

## Mac AMD64
$ENV:GOOS = "darwin"; $env:GOARCH = "amd64"
go build -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" -o .\bin\rmstale-mac_amd64 rmstale.go