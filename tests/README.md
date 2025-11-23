# Testing Structure

This directory contains two types of tests:

## Integration Tests (`tests/integration/`)

**Build tag**: `integration`

**Technology**: SQLite (in-memory) + httptest

**Purpose**: Fast integration tests that verify business logic and API contracts without external dependencies.

**Characteristics**:
- Use in-memory SQLite database
- Use `httptest.ResponseRecorder` for HTTP testing
- Fast execution (no Docker required)
- Test individual modules and their interactions
- Good for CI/CD pipelines where speed matters

**Run**: `go test -tags=integration ./tests/integration/...`

**Files**:
- `pullrequest_test.go` - PR lifecycle and reviewer assignment tests
- `team_test.go` - Team management tests
- `user_test.go` - User activity and review list tests

## E2E Tests (`tests/e2e/`)

**Build tag**: `e2e`

**Technology**: testcontainers-go + PostgreSQL 12 + Real HTTP server

**Purpose**: End-to-end tests that verify the entire system works correctly in production-like conditions.

**Characteristics**:
- Use real PostgreSQL 12 database via Docker containers
- Use real HTTP server (full application stack)
- Test migrations, constraints, triggers, indexes
- Test concurrent operations and race conditions
- Test PostgreSQL-specific features
- Require Docker daemon to be running

**Run**: `go test -tags=e2e ./tests/e2e/... -timeout 15m`

**Prerequisites**:
- Docker daemon must be running
- Sufficient resources for Docker containers

**Files**:
- `setup_test.go` - Test infrastructure (containers, DB, HTTP server)
- `business_scenarios_test.go` - Core business scenarios (scenarios 1-3)
- `error_scenarios_test.go` - Error handling scenarios (scenarios 4-6)
- `advanced_scenarios_test.go` - Advanced scenarios (scenarios 7-10)
- `edge_cases_test.go` - Edge cases and boundary conditions

## Test Scenarios Coverage

### Business Scenarios (E2E)
1. **Full PR Lifecycle** - Create team → Create PR → Auto-assign → Reassign → Merge → Idempotency
2. **Activity Management** - Inactive users not assigned, reactivation works
3. **Reviewer Count Limits** - Teams with 1, 2, 3+ members, inactive members

### Error Scenarios (E2E)
4. **NO_CANDIDATE** - Reassignment when no candidates available
5. **NOT_ASSIGNED** - Reassigning non-assigned reviewer
6. **Multiple PRs & getReview** - List all PRs for a reviewer (OPEN and MERGED)

### Advanced Scenarios (E2E)
7. **Concurrent PR Creation** - Race conditions, fair distribution
8. **Merge Idempotency** - Deep idempotency checks (timestamps, reviewers)
9. **Duplicate Keys** - TEAM_EXISTS, PR_EXISTS errors
10. **NOT_FOUND Errors** - All endpoints with non-existent entities

### Edge Cases (E2E)
- Unicode and special characters (Cyrillic, Japanese, Chinese)
- Reassignment chains
- Team immutability
- Long names (255 char limit)
- Empty reviewer lists

## Running Tests

### Run all integration tests:
```bash
go test -tags=integration ./tests/integration/... -v
```

### Run all E2E tests (requires Docker):
```bash
go test -tags=e2e ./tests/e2e/... -v -timeout 15m
```

### Run specific test suite:
```bash
go test -tags=e2e ./tests/e2e/... -run TestBusinessScenarios -v
```

### Run tests with coverage:
```bash
go test -tags=integration ./tests/integration/... -cover
go test -tags=e2e ./tests/e2e/... -cover -timeout 15m
```

## Differences

| Aspect | Integration Tests | E2E Tests |
|--------|------------------|-----------|
| Database | SQLite (in-memory) | PostgreSQL 12 (Docker) |
| HTTP | httptest | Real HTTP server |
| Speed | Fast (~seconds) | Slower (~minutes) |
| Dependencies | None | Docker required |
| Migrations | AutoMigrate | Real migrations |
| Constraints | Basic | Full PostgreSQL |
| Concurrency | Limited | Full testing |
| Use Case | CI/CD, fast feedback | Pre-release validation |

## Best Practices

1. **Integration tests** should be run frequently (on every commit)
2. **E2E tests** should be run before releases and in nightly builds
3. Both test suites should pass before merging PRs
4. E2E tests require Docker, so they may be skipped in some CI environments
5. Integration tests provide fast feedback during development

