# Write-Host "Getting GitVersion"
# choco install gitversion.portable -pre -y

Write-Host 'Generating version info using GitVersion'
$VersionInfo = gitversion | ConvertFrom-Json

Write-Host ("Version detected as '{0}'" -f $VersionInfo.SemVer)

# Build binaries
## Win x86
$ENV:GOOS = 'windows'; $env:GOARCH = '386'
go build -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" -o .\bin\windows\rmstale.exe rmstale.go

## Linux x86
$ENV:GOOS = 'linux'; $env:GOARCH = '386'
go build -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" -o .\bin\linux\rmstale rmstale.go

## Mac x86
$ENV:GOOS = 'darwin'; $env:GOARCH = 'amd64'
go build -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" -o .\bin\darwin\rmstale rmstale.go

$ENV:GOOS = 'windows'; $env:GOARCH = 'amd64'