# Write-Host "Getting GitVersion"
# choco install gitversion.portable -pre -y

Write-Host "Generating version info using GitVersion"
$VersionInfo = gitversion | convertfrom-json

Write-Host ("Version detected as '{0}'" -f $VersionInfo.SemVer)

go build -race -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" rmstale.go