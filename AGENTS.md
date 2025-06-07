# Project Overview

This repository contains `rmstale`, a small Go command line tool that removes stale files from a directory tree.

## Coding Style

- Use Go 1.19 or newer.
- Format all Go code with `gofmt -w` before committing.
- Check code with `golangci-lint` to ensure code quality.
- Stick to the standard Go style and keep the code cross-platform.
- Keep indentation with tabs for Go code (per `.editorconfig`). Use two spaces for YAML/JSON/Markdown.
- Follow existing commit message convention (`type: description`).

## Testing Style

- Ensure `go test ./...` runs successfully before submitting changes.
- Add or update tests whenever modifying code.
- Use the standard Go testing framework and `testify/suite` where appropriate.
