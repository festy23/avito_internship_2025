.PHONY: lint lint-fix test test-coverage test-verbose test-e2e test-integration test-coverage-show ci ci-local ci-local-lint ci-local-test ci-local-e2e ci-act ci-act-lint ci-act-test ci-act-e2e ci-act-list ci-local-clean

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

test-load:
	go test -tags=load ./tests/load/... -v -timeout 5m

ci: lint test-integration test
	@echo "All CI checks passed!"

ci-local:
	@echo "Running local CI checks (lint, integration tests, unit tests)..."
	$(MAKE) ci

ci-local-lint:
	@echo "Running lint locally..."
	$(MAKE) lint

ci-local-test:
	@echo "Running integration and unit tests locally..."
	$(MAKE) test-integration test

ci-local-e2e:
	@echo "Running E2E tests locally..."
	$(MAKE) test-e2e

ci-act:
	@echo "Running CI locally with act..."
	@echo "Note: This runs all jobs. Use specific targets (ci-act-lint, ci-act-test, ci-act-e2e) to run individual jobs."
	@act push -W .github/workflows/ci.yml --eventpath .github/workflows/ci-local.json --container-architecture linux/amd64 --rm

ci-act-lint:
	@echo "Running lint job locally with act..."
	@timeout 300 act -j lint -W .github/workflows/ci.yml --container-architecture linux/amd64 --rm -v || \
	 (echo "⚠ act failed or timed out, running lint directly..." && $(MAKE) lint)

ci-act-test:
	@echo "Running test job locally with act..."
	@act -j test -W .github/workflows/ci.yml --container-architecture linux/amd64 --rm || \
	 (echo "⚠ act failed, running tests directly..." && $(MAKE) test-integration test)

ci-act-e2e:
	@echo "Running E2E test job locally with act..."
	@act push -j test-e2e -W .github/workflows/ci.yml --eventpath .github/workflows/ci-local.json --container-architecture linux/amd64 --rm || \
	 (echo "⚠ act failed, running E2E tests directly..." && $(MAKE) test-e2e)

ci-act-list:
	@echo "Listing available CI jobs..."
	@act --list -W .github/workflows/ci.yml --container-architecture linux/amd64

ci-local-clean:
	@echo "Cleaning up act containers and images..."
	@docker ps -a --filter "ancestor=catthehacker/ubuntu:act-latest" --format "{{.ID}}" | xargs -r docker rm -f 2>/dev/null || true
	@docker system prune -f --filter "label=com.github.actions" 2>/dev/null || true
	@echo "✓ Cleanup complete"

