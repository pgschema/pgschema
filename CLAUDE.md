# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

pgschema is a simple CLI tool with version information.

## Commands

### Build
```bash
# Install from GitHub
go install github.com/pgschema/pgschema@latest

# Build locally
go build -o pgschema .
```

### Test
```bash
# Run all tests
go test -v ./...
```

### Dependencies
```bash
go mod tidy
```

## Architecture

The application uses:
- **Cobra** for CLI structure with `version` subcommand

Key components:
- `main.go`: Entry point that delegates to cmd package
- `cmd/`: Command implementations using Cobra
  - `cmd/root.go`: Root command and CLI setup
  - `cmd/version.go`: Version command
- Test files:
  - `main_test.go`: Tests for main package
  - `cmd/*_test.go`: Unit tests for each command