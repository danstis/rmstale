# Write-Host "Getting GitVersion"
# choco install gitversion.portable -pre -y

Write-Host "Generating version info using GitVersion"
gitversion /l console /output buildserver
$VersionInfo = gitversion | convertfrom-json

Write-Host ("Version detected as '{0}'" -f $VersionInfo.SemVer)

Write-Host ("Running: go build -race -ldflags `"-X main.AppVersion=$($VersionInfo.SemVer)`" rmstale.go")
go build -race -ldflags "-X main.AppVersion=$($VersionInfo.SemVer)" rmstale.go