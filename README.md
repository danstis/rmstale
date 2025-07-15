# rmstale

[![Build](https://github.com/danstis/rmstale/workflows/Build/badge.svg)](https://github.com/danstis/rmstale/actions?query=workflow%3ABuild)
[![Chocolatey](https://img.shields.io/chocolatey/v/rmstale.svg)](https://chocolatey.org/packages/rmstale)
[![Winget Version](https://img.shields.io/winget/v/danstis.rmstale)](https://winstall.app/apps/danstis.rmstale)
[![codecov](https://codecov.io/gh/danstis/rmstale/branch/master/graph/badge.svg)](https://codecov.io/gh/danstis/rmstale)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=danstis_rmstale&metric=alert_status)](https://sonarcloud.io/dashboard?id=danstis_rmstale)

rmstale is a tool to remove stale files recursively below a given directory.
Files and folders older than a defined period are removed.
A file is considered stale if it has not been modified in the last N days, where N is the value provided for the `--age` flag.
This tool will also remove directories that are considered stale (older than the defined period) and are empty.

Some examples for use:

* Set on a schedule to clear old files from your temporary directories.
* Set on a schedule to remove downloaded files from your downloads directory.

## Install instructions

### Install with Chocolatey

`choco install rmstale`

### Install with Winget

`winget install danstis.rmstale`

### Install on Linux

Visit the releases page to find the [latest release](https://github.com/danstis/rmstale/releases/latest) version.

```bash
# Fetch the latest release tag from GitHub
latest_version=$(curl -s https://api.github.com/repos/danstis/rmstale/releases/latest | grep -Po '"tag_name": "\K.*?(?=")')
# Download the latest version tarball
curl -L -o rmstale.tar.gz "https://github.com/danstis/rmstale/releases/download/$latest_version/rmstale_${latest_version#v}_linux_amd64.tar.gz"
# Extract and install
sudo tar -xzf rmstale.tar.gz -C /usr/bin rmstale
# Cleanup
rm rmstale.tar.gz
```

### Install rmstale manually

1. From the [GitHub releases page](https://github.com/danstis/rmstale/releases) download the latest binary for your operating system.
2. Place the downloaded file into your desired location.

## Usage instructions

### Command line flags

| Flag            | Description                                                                                                        | Default         |
| --------------- | ------------------------------------------------------------------------------------------------------------------ | --------------- |
| -a, --age       | Period in days before an item is considered stale. (REQUIRED)                                                      | *(required)*    |
| -d, --dry-run   | Runs the process in dry-run mode. No files will be removed, but the tool will log the files that would be deleted. | `false`         |
| -e, --extension | Filter files for a defined file extension. This flag only applies to files, not directories.                       | *(empty)*       |
| -p, --path      | Path to a folder to process.                                                                                       | system temp dir |
| -v, --version   | Displays the version of rmstale that is currently running.                                                         | `false`         |
| -y, --confirm   | Allows for processing without confirmation prompt, useful for scheduling.                                          | `false`         |

### Usage examples

```cmd
>: rmstale --version

rmstale v1.2.3
```

```cmd
>: rmstale --age 14 --path c:\temp
WARNING: Will remove files and folders recursively below 'c:\temp' older than 14 days. Continue?: y

-Removing 'C:\Temp\amc2E40.tmp.LOG1'...
-Removing 'C:\Temp\amc2E40.tmp.LOG2'...
-Removing 'C:\Temp\amc306D.tmp.LOG1'...
-Removing 'C:\Temp\amc306D.tmp.LOG2'...
-Removing 'C:\Temp\amc308D.tmp.LOG1'...
```

Any errors encountered during the deletion process (e.g., permission issues) will be logged.

## GitHub project

Feedback, Issues, Bugs and Contribution to this tool are welcome.
For Bugs/Issues/Feature requests, please create an issue on the [GitHub issues page](https://github.com/danstis/rmstale/issues).

Want to contribute? Great:

* Fork the repo using the Fork button at the top right of the GitHub repo.
* Clone the repo to your development machine, note the dependencies for this project are as follows:
  * Go version 1.24 or above
* Create a new branch for the feature that you want to contribute.
* Develop your new feature as you see fit.
* Once you have a working copy of your code, create a pull request against this project.

### Release Process

This project follows semantic versioning (SemVer) for releases. To create a new release:

1. Ensure all changes are committed to the main branch via a pull request.
2. The release pipeline will automatically generate a tag if the build number is a clean semver version without a prerelease tag (e.g., `v1.2.3`).
3. Alternatively, you can manually tag a commit with a version number following the format `v1.2.3` (where 1 is the major version, 2 is the minor version, and 3 is the patch version) and push the tag to the repository.
4. The CI/CD pipeline will automatically build and publish the release.
