# Cloud Requirements

## Setup
- Go toolchain 1.24+ (auto-download via `go` toolchain).
- Docker is required for Kafka integration tests.

## Minimal Checks
- `go test ./...`
- `go test -tags kafka ./infra/eventbus` (requires Docker)
- `golangci-lint run`

## Notes
- If Docker is unavailable, skip the Kafka-tagged test and note it in CI output.
