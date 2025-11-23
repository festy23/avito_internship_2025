//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/client"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresDriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	pullrequestModel "github.com/festy23/avito_internship/internal/pullrequest/model"
	teamModel "github.com/festy23/avito_internship/internal/team/model"
	userModel "github.com/festy23/avito_internship/internal/user/model"
)

// E2ETestSuite contains test infrastructure
type E2ETestSuite struct {
	suite.Suite
	ctx              context.Context
	pgContainer      *postgres.PostgresContainer
	db               *gorm.DB
	appContainer     testcontainers.Container
	baseURL          string
	httpClient       *http.Client
	connectionString string
}

// SetupSuite runs once before all tests
func (s *E2ETestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Start PostgreSQL container
	pgContainer, err := postgres.Run(s.ctx,
		"postgres:12-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(s.T(), err, "failed to start PostgreSQL container")
	s.pgContainer = pgContainer

	// Get connection string
	connStr, err := pgContainer.ConnectionString(s.ctx, "sslmode=disable")
	require.NoError(s.T(), err, "failed to get connection string")
	s.connectionString = connStr

	// Connect to database (for test assertions only)
	// Migrations will be applied by the application container on startup
	// The migrate.Up() function handles ErrNoChange, so it's safe to call multiple times
	db, err := gorm.Open(postgresDriver.Open(connStr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(s.T(), err, "failed to connect to database")
	s.db = db

	// Note: Do NOT apply migrations here - let the application container do it
	// This tests the real migration path and ensures migrations work correctly

	// Get PostgreSQL container's internal IP address for inter-container communication
	// We need the internal IP, not the mapped host/port
	containerName, err := pgContainer.Name(s.ctx)
	require.NoError(s.T(), err, "failed to get PostgreSQL container name")

	// Get Docker client to inspect container network settings
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	require.NoError(s.T(), err, "failed to create Docker client")
	defer dockerClient.Close()

	// Inspect container by name to get network settings
	// Remove leading "/" from container name for Docker API
	containerNameClean := strings.TrimPrefix(containerName, "/")
	containerInfo, err := dockerClient.ContainerInspect(s.ctx, containerNameClean)
	require.NoError(s.T(), err, "failed to inspect PostgreSQL container")

	// Get the first network's IP address (containers are typically on one network)
	var dbHost string
	var dbPort = "5432"
	if len(containerInfo.NetworkSettings.Networks) > 0 {
		// Get IP address from the first network
		for _, network := range containerInfo.NetworkSettings.Networks {
			dbHost = network.IPAddress
			break
		}
	}

	// Fallback to container name if IP not found
	if dbHost == "" {
		dbHost = containerNameClean
	}

	// Start application container
	// testcontainers-go should place containers in the same network
	// Use the hostname/IP from connection string for inter-container communication
	// Use pre-built image to avoid rebuilding for each test suite
	appContainer, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "avito-internship-e2e:test",
			ExposedPorts: []string{"8080/tcp"},
			Env: map[string]string{
				"DB_HOST":                dbHost, // Use hostname/IP from connection string
				"DB_PORT":                dbPort, // Use port from connection string
				"DB_USER":                "testuser",
				"DB_PASSWORD":            "testpass",
				"DB_NAME":                "testdb",
				"DB_SSLMODE":             "disable",
				"DB_TIMEZONE":            "UTC",
				"DB_RETRY_MAX_ATTEMPTS":  "5",
				"DB_RETRY_INITIAL_DELAY": "1s",
				"DB_RETRY_MAX_DELAY":     "30s",
				"DB_RETRY_MULTIPLIER":    "2.0",
				"SERVER_HOST":            "",
				"SERVER_PORT":            ":8080",
				"SERVER_READ_TIMEOUT":    "10s",
				"SERVER_WRITE_TIMEOUT":   "10s",
				"SERVER_IDLE_TIMEOUT":    "120s",
				"GIN_MODE":               "release",
				"LOG_LEVEL":              "info",
				"LOG_FORMAT":             "json",
				"LOG_OUTPUT":             "stdout",
				"MIGRATIONS_PATH":        "migrations",
			},
			WaitingFor: wait.ForHTTP("/health").
				WithPort("8080/tcp").
				WithStartupTimeout(120 * time.Second).
				WithPollInterval(2 * time.Second),
		},
		Started: true,
	})
	require.NoError(s.T(), err, "failed to start application container")
	s.appContainer = appContainer

	// Get application URL
	host, err := appContainer.Host(s.ctx)
	require.NoError(s.T(), err, "failed to get container host")

	port, err := appContainer.MappedPort(s.ctx, "8080")
	require.NoError(s.T(), err, "failed to get container port")

	s.baseURL = fmt.Sprintf("http://%s:%s", host, port.Port())
	s.httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}

	// Wait for application to be ready
	s.waitForApp()

	// Log configuration for debugging
	s.logConfiguration()

	// Verify migrations were applied by checking if tables exist
	s.verifyMigrations()

	// Log application startup logs for debugging
	s.logAppStartup()
}

// TearDownSuite runs once after all tests
func (s *E2ETestSuite) TearDownSuite() {
	if s.appContainer != nil {
		_ = s.appContainer.Terminate(s.ctx)
	}
	if s.pgContainer != nil {
		_ = s.pgContainer.Terminate(s.ctx)
	}
}

// SetupTest runs before each test
func (s *E2ETestSuite) SetupTest() {
	// Clean all tables
	s.cleanDatabase()
}

// applyMigrations applies database migrations
func (s *E2ETestSuite) applyMigrations() {
	// Create tables in order
	migrations := []string{
		// teams table
		`CREATE TABLE IF NOT EXISTS teams (
			team_name VARCHAR(255) PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT chk_team_name_length CHECK (LENGTH(team_name) BETWEEN 1 AND 255)
		)`,
		// updated_at trigger function
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,
		// teams trigger
		`DROP TRIGGER IF EXISTS trigger_teams_updated_at ON teams`,
		`CREATE TRIGGER trigger_teams_updated_at
			BEFORE UPDATE ON teams
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column()`,
		// users table
		`CREATE TABLE IF NOT EXISTS users (
			user_id VARCHAR(255) PRIMARY KEY,
			username VARCHAR(255) NOT NULL,
			team_name VARCHAR(255) NOT NULL,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_users_team_name FOREIGN KEY (team_name) 
				REFERENCES teams(team_name) ON DELETE RESTRICT,
			CONSTRAINT chk_user_id_length CHECK (LENGTH(user_id) BETWEEN 1 AND 255),
			CONSTRAINT chk_username_length CHECK (LENGTH(username) BETWEEN 1 AND 255),
			CONSTRAINT chk_team_name_length CHECK (LENGTH(team_name) BETWEEN 1 AND 255)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_users_team_name ON users(team_name)`,
		`CREATE INDEX IF NOT EXISTS idx_users_team_active ON users(team_name, is_active)`,
		`DROP TRIGGER IF EXISTS trigger_users_updated_at ON users`,
		`CREATE TRIGGER trigger_users_updated_at
			BEFORE UPDATE ON users
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column()`,
		// pull_requests table
		`CREATE TABLE IF NOT EXISTS pull_requests (
			pull_request_id VARCHAR(255) PRIMARY KEY,
			pull_request_name VARCHAR(255) NOT NULL,
			author_id VARCHAR(255) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'OPEN',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			merged_at TIMESTAMPTZ,
			CONSTRAINT fk_pull_requests_author_id FOREIGN KEY (author_id) 
				REFERENCES users(user_id) ON DELETE RESTRICT,
			CONSTRAINT chk_pull_request_id_length CHECK (LENGTH(pull_request_id) BETWEEN 1 AND 255),
			CONSTRAINT chk_pull_request_name_length CHECK (LENGTH(pull_request_name) BETWEEN 1 AND 255),
			CONSTRAINT chk_status CHECK (status IN ('OPEN', 'MERGED'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pull_requests_author_id ON pull_requests(author_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pull_requests_status ON pull_requests(status)`,
		// pull_request_reviewers table
		`CREATE TABLE IF NOT EXISTS pull_request_reviewers (
			id SERIAL PRIMARY KEY,
			pull_request_id VARCHAR(255) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_pr_reviewers_pull_request_id FOREIGN KEY (pull_request_id) 
				REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
			CONSTRAINT fk_pr_reviewers_user_id FOREIGN KEY (user_id) 
				REFERENCES users(user_id) ON DELETE RESTRICT,
			CONSTRAINT uq_pr_reviewer UNIQUE (pull_request_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_pr_reviewers_pull_request_id ON pull_request_reviewers(pull_request_id)`,
		`CREATE INDEX IF NOT EXISTS idx_pr_reviewers_user_id ON pull_request_reviewers(user_id)`,
	}

	for _, migration := range migrations {
		err := s.db.Exec(migration).Error
		require.NoError(s.T(), err, "failed to apply migration")
	}
}

// cleanDatabase truncates all tables
func (s *E2ETestSuite) cleanDatabase() {
	s.db.Exec("TRUNCATE TABLE pull_request_reviewers CASCADE")
	s.db.Exec("TRUNCATE TABLE pull_requests CASCADE")
	s.db.Exec("TRUNCATE TABLE users CASCADE")
	s.db.Exec("TRUNCATE TABLE teams CASCADE")
}

// waitForApp waits for the application to be ready
func (s *E2ETestSuite) waitForApp() {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := s.httpClient.Get(s.baseURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(1 * time.Second)
	}
	s.T().Fatal("application did not become ready in time")
}

// Helper methods for HTTP requests

// doRequest performs HTTP request and returns response
func (s *E2ETestSuite) doRequest(method, path string, body io.Reader) (*http.Response, []byte) {
	req, err := http.NewRequest(method, s.baseURL+path, body)
	require.NoError(s.T(), err, "failed to create request")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	require.NoError(s.T(), err, "failed to perform request")

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(s.T(), err, "failed to read response body")
	resp.Body.Close()

	return resp, respBody
}

// doRequestNoFail performs HTTP request and returns response with error.
// Safe to use in goroutines as it doesn't call require/assert.
func (s *E2ETestSuite) doRequestNoFail(method, path string, body io.Reader) (*http.Response, []byte, error) {
	req, err := http.NewRequest(method, s.baseURL+path, body)
	if err != nil {
		return nil, nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	respBody, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return resp, nil, err
	}

	return resp, respBody, nil
}

// createTeam creates a team via HTTP API
func (s *E2ETestSuite) createTeam(req *teamModel.AddTeamRequest) (*http.Response, *teamModel.TeamResponse) {
	bodyBytes, _ := json.Marshal(req)
	resp, respBody := s.doRequest("POST", "/team/add", strings.NewReader(string(bodyBytes)))

	if resp.StatusCode != http.StatusCreated {
		s.T().Logf("❌ Failed to create team")
		s.T().Logf("   Status Code: %d", resp.StatusCode)
		s.T().Logf("   Request Body: %s", string(bodyBytes))
		s.T().Logf("   Response Body: %s", string(respBody))

		// Get application logs for debugging
		appLogs := s.getAppLogs()
		if appLogs != "" {
			s.T().Logf("   Application Logs (last 20 lines):")
			lines := strings.Split(appLogs, "\n")
			start := 0
			if len(lines) > 20 {
				start = len(lines) - 20
			}
			for i := start; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) != "" {
					s.T().Logf("   %s", lines[i])
				}
			}
		}
		return resp, nil
	}

	var result struct {
		Team teamModel.TeamResponse `json:"team"`
	}
	err := json.Unmarshal(respBody, &result)
	require.NoError(s.T(), err, "failed to unmarshal team response")

	return resp, &result.Team
}

// getTeam gets a team via HTTP API
func (s *E2ETestSuite) getTeam(teamName string) (*http.Response, *teamModel.TeamResponse) {
	resp, respBody := s.doRequest("GET", fmt.Sprintf("/team/get?team_name=%s", teamName), nil)

	if resp.StatusCode != http.StatusOK {
		return resp, nil
	}

	var result teamModel.TeamResponse
	err := json.Unmarshal(respBody, &result)
	require.NoError(s.T(), err, "failed to unmarshal team response")

	return resp, &result
}

// setUserActive sets user active status via HTTP API
func (s *E2ETestSuite) setUserActive(userID string, isActive bool) (*http.Response, *userModel.User) {
	req := userModel.SetIsActiveRequest{
		UserID:   userID,
		IsActive: isActive,
	}
	bodyBytes, _ := json.Marshal(req)
	resp, respBody := s.doRequest("POST", "/users/setIsActive", strings.NewReader(string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		return resp, nil
	}

	var result struct {
		User userModel.User `json:"user"`
	}
	err := json.Unmarshal(respBody, &result)
	require.NoError(s.T(), err, "failed to unmarshal user response")

	return resp, &result.User
}

// createPR creates a pull request via HTTP API
func (s *E2ETestSuite) createPR(req *pullrequestModel.CreatePullRequestRequest) (*http.Response, *pullrequestModel.PullRequestResponse) {
	bodyBytes, _ := json.Marshal(req)
	resp, respBody := s.doRequest("POST", "/pullRequest/create", strings.NewReader(string(bodyBytes)))

	if resp.StatusCode != http.StatusCreated {
		s.T().Logf("❌ Failed to create PR")
		s.T().Logf("   Status Code: %d", resp.StatusCode)
		s.T().Logf("   Request Body: %s", string(bodyBytes))
		s.T().Logf("   Response Body: %s", string(respBody))

		// Get application logs for debugging
		appLogs := s.getAppLogs()
		if appLogs != "" {
			s.T().Logf("   Application Logs (last 10 lines):")
			lines := strings.Split(appLogs, "\n")
			start := 0
			if len(lines) > 10 {
				start = len(lines) - 10
			}
			for i := start; i < len(lines); i++ {
				if strings.TrimSpace(lines[i]) != "" {
					s.T().Logf("   %s", lines[i])
				}
			}
		}
		return resp, nil
	}

	var result struct {
		PR pullrequestModel.PullRequestResponse `json:"pr"`
	}
	err := json.Unmarshal(respBody, &result)
	require.NoError(s.T(), err, "failed to unmarshal PR response")

	return resp, &result.PR
}

// createPRNoFail creates a pull request via HTTP API and returns error.
// Safe to use in goroutines as it doesn't call require/assert.
func (s *E2ETestSuite) createPRNoFail(req *pullrequestModel.CreatePullRequestRequest) (*http.Response, *pullrequestModel.PullRequestResponse, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, nil, err
	}

	resp, respBody, err := s.doRequestNoFail("POST", "/pullRequest/create", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusCreated {
		return resp, nil, nil
	}

	var result struct {
		PR pullrequestModel.PullRequestResponse `json:"pr"`
	}
	err = json.Unmarshal(respBody, &result)
	if err != nil {
		return resp, nil, err
	}

	return resp, &result.PR, nil
}

// mergePR merges a pull request via HTTP API
func (s *E2ETestSuite) mergePR(prID string) (*http.Response, *pullrequestModel.PullRequestResponse) {
	req := pullrequestModel.MergePullRequestRequest{
		PullRequestID: prID,
	}
	bodyBytes, _ := json.Marshal(req)
	resp, respBody := s.doRequest("POST", "/pullRequest/merge", strings.NewReader(string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		return resp, nil
	}

	var result struct {
		PR pullrequestModel.PullRequestResponse `json:"pr"`
	}
	err := json.Unmarshal(respBody, &result)
	require.NoError(s.T(), err, "failed to unmarshal PR response")

	return resp, &result.PR
}

// reassignReviewer reassigns a reviewer via HTTP API
// Returns response and response body bytes for error parsing
func (s *E2ETestSuite) reassignReviewer(prID, oldUserID string) (*http.Response, *pullrequestModel.ReassignReviewerResponse, []byte) {
	req := pullrequestModel.ReassignReviewerRequest{
		PullRequestID: prID,
		OldUserID:     oldUserID,
	}
	bodyBytes, _ := json.Marshal(req)
	resp, respBody := s.doRequest("POST", "/pullRequest/reassign", strings.NewReader(string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		return resp, nil, respBody
	}

	var result pullrequestModel.ReassignReviewerResponse
	err := json.Unmarshal(respBody, &result)
	require.NoError(s.T(), err, "failed to unmarshal reassign response")

	return resp, &result, respBody
}

// getUserReviews gets user's assigned PRs via HTTP API
func (s *E2ETestSuite) getUserReviews(userID string) (*http.Response, *userModel.GetReviewResponse) {
	resp, respBody := s.doRequest("GET", fmt.Sprintf("/users/getReview?user_id=%s", userID), nil)

	if resp.StatusCode != http.StatusOK {
		return resp, nil
	}

	var result userModel.GetReviewResponse
	err := json.Unmarshal(respBody, &result)
	require.NoError(s.T(), err, "failed to unmarshal review response")

	return resp, &result
}

// parseErrorResponse parses error response
func (s *E2ETestSuite) parseErrorResponse(respBody []byte) (string, string) {
	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	err := json.Unmarshal(respBody, &errResp)
	require.NoError(s.T(), err, "failed to unmarshal error response")
	return errResp.Error.Code, errResp.Error.Message
}

// Assertion helpers

// verifyMigrations checks if database migrations were applied successfully
func (s *E2ETestSuite) verifyMigrations() {
	s.T().Logf("=== Verifying Database Migrations ===")
	tables := []string{"teams", "users", "pull_requests", "pull_request_reviewers"}

	allExist := true
	for _, table := range tables {
		var exists bool
		err := s.db.Raw(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables 
				WHERE table_schema = 'public' 
				AND table_name = ?
			)`, table).Scan(&exists).Error

		if err != nil {
			s.T().Logf("❌ Failed to check if table %s exists: %v", table, err)
			allExist = false
			continue
		}

		if !exists {
			s.T().Logf("❌ Table %s does NOT exist - migrations may not have been applied", table)
			allExist = false
		} else {
			s.T().Logf("✓ Table %s exists", table)

			// Check table structure
			var count int64
			s.db.Table(table).Count(&count)
			s.T().Logf("  └─ Row count: %d", count)
		}
	}

	if !allExist {
		s.T().Logf("⚠️  Some tables are missing - checking application logs...")
		appLogs := s.getAppLogs()
		if appLogs != "" {
			s.T().Logf("Application logs (migration-related):")
			lines := strings.Split(appLogs, "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToLower(line), "migration") ||
					strings.Contains(strings.ToLower(line), "table") ||
					strings.Contains(strings.ToLower(line), "error") ||
					strings.Contains(strings.ToLower(line), "fatal") {
					s.T().Logf("  %s", line)
				}
			}
		}
	} else {
		s.T().Logf("✓ All migrations verified successfully")
	}
	s.T().Logf("====================================")
}

// logConfiguration logs important configuration values for debugging
func (s *E2ETestSuite) logConfiguration() {
	s.T().Logf("=== E2E Test Configuration ===")
	s.T().Logf("Application URL: %s", s.baseURL)
	s.T().Logf("Database connection: %s", s.connectionString)
	if s.appContainer != nil {
		host, _ := s.appContainer.Host(s.ctx)
		port, _ := s.appContainer.MappedPort(s.ctx, "8080")
		s.T().Logf("Container Host: %s, Port: %s", host, port.Port())
	}
	if s.pgContainer != nil {
		pgHost, _ := s.pgContainer.Host(s.ctx)
		pgPort, _ := s.pgContainer.MappedPort(s.ctx, "5432")
		s.T().Logf("PostgreSQL Host: %s, Port: %s", pgHost, pgPort.Port())
	}
	s.T().Logf("=============================")
}

// logAppStartup logs application container startup logs
func (s *E2ETestSuite) logAppStartup() {
	if s.appContainer == nil {
		return
	}

	logs := s.getAppLogs()
	if logs != "" {
		s.T().Logf("=== Application Startup Logs ===")
		// Show last 50 lines of logs
		lines := strings.Split(logs, "\n")
		start := 0
		if len(lines) > 50 {
			start = len(lines) - 50
		}
		for i := start; i < len(lines); i++ {
			if lines[i] != "" {
				s.T().Logf("%s", lines[i])
			}
		}
		s.T().Logf("================================")
	}
}

// getAppLogs retrieves application container logs
func (s *E2ETestSuite) getAppLogs() string {
	if s.appContainer == nil {
		return ""
	}

	logs, err := s.appContainer.Logs(s.ctx)
	if err != nil {
		return fmt.Sprintf("Failed to get logs: %v", err)
	}
	defer logs.Close()

	logBytes, err := io.ReadAll(logs)
	if err != nil {
		return fmt.Sprintf("Failed to read logs: %v", err)
	}

	return string(logBytes)
}
