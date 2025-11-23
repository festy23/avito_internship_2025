.PHONY: lint lint-fix test test-coverage test-verbose test-e2e test-integration test-coverage-show ci

lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

test:
	go test ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-coverage-show:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

test-verbose:
	go test -v ./...

test-e2e-build:
	docker build -t avito-internship-e2e:test -f Dockerfile .

test-e2e: test-e2e-build
	go test -tags=e2e ./tests/e2e/... -v -timeout 20m

test-integration:
	go test -tags=integration ./tests/integration/... -v

ci: lint test-integration test
	@echo "All CI checks passed!"

