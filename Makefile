.PHONY: lint lint-fix test test-coverage test-verbose test-e2e test-integration test-coverage-show

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

test-e2e:
	go test -tags=e2e ./tests/e2e/... -v -timeout 20m

test-integration:
	go test -tags=integration ./tests/integration/... -v

