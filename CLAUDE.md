# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Tegridy Tokens is a high-performance Go library for tokenizing and detokenizing sensitive data using AWS DynamoDB and KMS. It supports both standard and format-preserving tokenization for big data workflows.

## Go Development Commands

### Project Setup
```bash
go mod init github.com/username/tegridy-tokens  # Initialize Go module
go mod tidy                                      # Clean up dependencies
```

### Development
```bash
go run .                # Run the application
go run ./cmd/app        # Run specific cmd (if using cmd pattern)
go build -o tegridy     # Build binary
```

### Testing
```bash
go test ./...           # Run all tests
go test -v ./...        # Run tests with verbose output
go test -cover ./...    # Run tests with coverage
go test -race ./...     # Run tests with race detector
go test ./pkg/...       # Test specific package
```

### Code Quality
```bash
go fmt ./...            # Format code
go vet ./...            # Run static analysis
golangci-lint run       # Run linter (if installed)
```

## Project Structure

Recommended Go project layout:
```
tegridy-tokens/
├── cmd/                # Application entrypoints
│   └── server/         # Main application
├── pkg/                # Public libraries
├── api/                # API definitions (OpenAPI/Proto)
├── scripts/            # Build/deploy scripts
├── go.mod              # Go module file
├── go.sum              # Go module checksums
└── Makefile            # Build automation
```

## Development Workflow

1. Use `go mod` for dependency management
2. Follow standard Go project layout
3. Write tests alongside code (*_test.go files)
4. Use interfaces for testability and loose coupling
5. Handle errors explicitly - don't ignore them
6. Use context.Context for cancellation and timeouts

## Go Conventions

- Package names: lowercase, single word
- Exported names: Start with capital letter
- File names: lowercase with underscores
- Test files: *_test.go in same package
- Interfaces: Often end with "-er" suffix
- Error handling: Check and handle all errors

## Project-Specific Notes

### Testing
- Run integration tests with: `make test-integration`
- Integration tests require AWS credentials and real infrastructure
- Unit tests can be run with: `go test -v ./pkg/tokenizer/...`

### Performance
- Caching has been removed from the library - handle at infrastructure level (DAX, Redis, etc.)
- Use batch operations for high-throughput scenarios
- Configure worker pools appropriately for concurrent processing

### AWS Dependencies
- DynamoDB: Token storage with TTL support
- KMS: Envelope encryption for performance and security
- Ensure proper IAM permissions for DynamoDB and KMS operations