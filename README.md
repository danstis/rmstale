# rmstale

[![Build](https://github.com/danstis/rmstale/workflows/Build/badge.svg)](https://github.com/danstis/rmstale/actions?query=workflow%3ABuild)
[![Chocolatey](https://img.shields.io/chocolatey/v/rmstale.svg)](https://chocolatey.org/packages/rmstale)
[![Winget Version](https://img.shields.io/winget/v/danstis.rmstale)](https://winstall.app/apps/danstis.rmstale)
[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=danstis_rmstale&metric=coverage)](https://sonarcloud.io/summary/new_code?id=danstis_rmstale)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=danstis_rmstale&metric=alert_status)](https://sonarcloud.io/dashboard?id=danstis_rmstale)
[![Go Version](https://img.shields.io/badge/go-1.25%2B-blue)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE.txt)

> **rmstale** is a cross-platform command-line tool to remove stale files and empty directories recursively below a given directory.

---

## Table of Contents
- [Features](#features)
- [Install instructions](#install-instructions)
- [Usage instructions](#usage-instructions)
- [Trust model / Security](#trust-model--security)
- [Contribution](#contribution)
- [Release Process](#release-process)
- [License](#license)

## Features
- Remove files and empty directories older than a specified age
- Cross-platform: Windows, Linux, macOS
- Dry-run mode for safe testing
- Extension-based file filtering
- Logs errors (e.g., permission issues)
- Designed for automation and scheduling

Some example use cases:
- Set on a schedule to clear old files from your temporary directories
- Set on a schedule to remove downloaded files from your downloads directory

## Install instructions

### Windows (Chocolatey)
```sh
choco install rmstale
```

### Windows (Winget)
```sh
winget install danstis.rmstale
```

### Linux
Visit the [releases page](https://github.com/danstis/rmstale/releases/latest) for the latest version, or use:
```sh
# Fetch the latest release tag from GitHub
latest_version=$(curl -s https://api.github.com/repos/danstis/rmstale/releases/latest | grep -Po '"tag_name": "\K.*?(?=")')
# Download the latest version tarball
curl -L -o rmstale.tar.gz "https://github.com/danstis/rmstale/releases/download/$latest_version/rmstale_${latest_version#v}_linux_amd64.tar.gz"
# Extract and install
sudo tar -xzf rmstale.tar.gz -C /usr/local/bin rmstale
# Cleanup
rm rmstale.tar.gz
```

### macOS
Visit the [releases page](https://github.com/danstis/rmstale/releases/latest) for the latest version, or use the commands below.

#### Apple Silicon (M1/M2/M3/M4)
```sh
# Fetch the latest release tag from GitHub
latest_version=$(curl -s https://api.github.com/repos/danstis/rmstale/releases/latest | grep -Po '"tag_name": "\K.*?(?=")')
# Download the latest version tarball for Apple Silicon
curl -L -o rmstale.tar.gz "https://github.com/danstis/rmstale/releases/download/$latest_version/rmstale_${latest_version#v}_Darwin_arm64.tar.gz"
# Extract and install
sudo tar -xzf rmstale.tar.gz -C /usr/local/bin rmstale
# Cleanup
rm rmstale.tar.gz
```

#### Intel
```sh
# Fetch the latest release tag from GitHub
latest_version=$(curl -s https://api.github.com/repos/danstis/rmstale/releases/latest | grep -Po '"tag_name": "\K.*?(?=")')
# Download the latest version tarball for Intel
curl -L -o rmstale.tar.gz "https://github.com/danstis/rmstale/releases/download/$latest_version/rmstale_${latest_version#v}_Darwin_x86_64.tar.gz"
# Extract and install
sudo tar -xzf rmstale.tar.gz -C /usr/local/bin rmstale
# Cleanup
rm rmstale.tar.gz
```

### Manual Download
1. Download the latest binary for your OS from the [GitHub releases page](https://github.com/danstis/rmstale/releases).
2. Place the downloaded file into a directory in your PATH.

## Usage instructions

### Command Line Flags

| Flag                 | Description                                                                                                                                                              | Default         |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | --------------- |
| -a, --age            | Period in days before an item is considered stale. (REQUIRED)                                                                                                            | *(required)*    |
| -d, --dry-run        | Runs the process in dry-run mode. No files will be removed, but the tool will log the files that would be deleted.                                                       | `false`         |
| -e, --extension      | Filter files for a defined file extension. This flag only applies to files, not directories.                                                                             | *(empty)*       |
| -p, --path           | Path to a folder to process. Resolved to an absolute, cleaned path before use. Refused when it equals a protected filesystem root (see `--allow-system-paths`).          | system temp dir |
| --allow-system-paths | Permit rmstale to operate on protected system roots (`/`, `C:\`, `/etc`, `/usr`, `/var`, `/boot`, `/proc`, `/sys`, or the current user's home directory). Sub-paths remain reachable by default. | `false`         |
| --prune-empty-dirs   | Remove empty directories even if they are not stale.                                                                                                                     | `false`         |
| -v, --version        | Displays the version of rmstale that is currently running.                                                                                                               | `false`         |
| -y, --confirm        | Allows for processing without confirmation prompt, useful for scheduling.                                                                                                | `false`         |

#### Safety: protected paths

`rmstale` refuses to descend into exact-match protected filesystem roots unless
`--allow-system-paths` is set. The protected roots are:

- `/` (filesystem root on Linux/macOS) and `C:\` (drive root on Windows)
- `/etc`, `/usr`, `/var`, `/boot`, `/proc`, `/sys` (Linux/macOS system roots)
- the current user's home directory (`os.UserHomeDir()`)

Sub-paths of those roots — for example `/var/tmp`, `/home/user/data`, or
`C:\Users\<user>\AppData\Local\Temp` — remain reachable without the override so
legitimate staging areas keep working out of the box. Scheduled jobs that
genuinely need to clean files inside a protected root should pass
`--allow-system-paths` after reviewing the consequences (a typo in the path
could destroy system state).

### Usage Examples

#### Show Version
```sh
rmstale --version
# rmstale v1.2.3
```

#### Remove files older than 14 days (Windows)
```sh
rmstale --age 14 --path c:\temp
# WARNING: Will remove files and folders recursively below 'c:\temp' older than 14 days. Continue?: y
# -Removing 'C:\Temp\amc2E40.tmp.LOG1'...
# ...
```

#### Remove files older than 14 days (Linux/macOS)
```sh
rmstale --age 14 --path /tmp
# WARNING: Will remove files and folders recursively below '/tmp' older than 14 days. Continue?: y
# -Removing '/tmp/oldfile1'...
# ...
```

#### Dry-run mode (no files deleted)
```sh
rmstale --age 30 --path ~/Downloads --dry-run
# [DRY RUN] '/home/user/Downloads/oldfile.txt' would be removed...
# ...
```

Any errors encountered during the deletion process (e.g., permission issues) will be logged.

## Trust model / Security

rmstale decides whether a file or directory is stale from its modification time (`ModTime`) as reported by the operating system. This metadata is **trusted but not verified**: on every supported platform the file owner can change it freely using tools like `touch -d`, `os.Chtimes`, PowerShell's `LastWriteTime`, or the file properties dialog. Nothing rmstale reads about a file is cryptographically signed or tamper-evident.

In practice this means:

- A file that should be removed can be **pinned** past the cleanup window by setting its `ModTime` to the future (e.g. `touch -d "2099-01-01" file`).
- Conversely, a file can be **forced into deletion** by back-dating its `ModTime`, which also lets an empty parent directory be pruned on the next run.

Recommendations for operators running rmstale against shared or untrusted directories (CI runners, scratch storage, multi-tenant `/tmp` directories, HPC workspaces, etc.):

- Run rmstale as a **single, dedicated user** that owns the directories being cleaned up. Treat that user's `ModTime` writes as the only ones you trust.
- Run rmstale from a **privileged context** (e.g. a scheduled task, cron job, or service account) so that any tampering by other users with write access to the metadata cannot bypass the policy.
- Avoid running rmstale against directories writable by users whose files you would not want to keep or delete; the `ModTime` it sees is the one those users set.
- Use `--dry-run` first when pointing rmstale at a new path to confirm the age threshold and ownership assumptions match your expectations.

## Contribution

Feedback, issues, and contributions are welcome! For bugs, issues, or feature requests, please create an issue on the [GitHub issues page](https://github.com/danstis/rmstale/issues).

Want to contribute? Please:
- Fork the repo and clone it locally
- Use Go 1.24 or above
- Format code with `gofmt -w` and check with `golangci-lint`
- Add or update tests (`go test ./...`)
- Follow the [CONTRIBUTING.md](CONTRIBUTING.md) guidelines
- Create a pull request with a clear description

## Release Process

This project follows [Semantic Versioning](https://semver.org/). To create a new release:
1. Ensure all changes are committed to the main branch via a pull request
2. The release pipeline will automatically generate a tag if the build number is a clean semver version (e.g., `v1.2.3`)
3. Alternatively, manually tag a commit with a version number (`v1.2.3`) and push the tag
4. The CI/CD pipeline will build and publish the release

## License

This project is licensed under the [MIT License](LICENSE.txt).
