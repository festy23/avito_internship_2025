.PHONY: lint lint-fix test

# Run linter
lint:
	golangci-lint run

# Run linter with auto-fix
lint-fix:
	golangci-lint run --fix

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Run tests with verbose output
test-verbose:
	go test -v ./...

# Run E2E tests
test-e2e:
	go test -tags=e2e ./tests/e2e/... -v

