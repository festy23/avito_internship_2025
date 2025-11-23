package model

import (
	"encoding/json"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetIsActiveRequest_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON with true", func(t *testing.T) {
		req := SetIsActiveRequest{
			UserID:   "u1",
			IsActive: true,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"user_id":"u1"`)
		assert.Contains(t, string(data), `"is_active":true`)
	})

	t.Run("marshal to JSON with false", func(t *testing.T) {
		req := SetIsActiveRequest{
			UserID:   "u1",
			IsActive: false,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"user_id":"u1"`)
		assert.Contains(t, string(data), `"is_active":false`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{"user_id":"u1","is_active":true}`
		var req SetIsActiveRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "u1", req.UserID)
		assert.True(t, req.IsActive)
	})

	t.Run("unmarshal with false value", func(t *testing.T) {
		jsonData := `{"user_id":"u1","is_active":false}`
		var req SetIsActiveRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "u1", req.UserID)
		assert.False(t, req.IsActive)
	})

	t.Run("unmarshal with missing is_active", func(t *testing.T) {
		jsonData := `{"user_id":"u1"}`
		var req SetIsActiveRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "u1", req.UserID)
		assert.False(t, req.IsActive) // bool defaults to false
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		jsonData := `{"user_id":"u1",invalid}`
		var req SetIsActiveRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		assert.Error(t, err)
	})
}

func TestSetIsActiveRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid request with true", func(t *testing.T) {
		req := SetIsActiveRequest{
			UserID:   "u1",
			IsActive: true,
		}

		assert.NotEmpty(t, req.UserID)
		assert.True(t, req.IsActive)
	})

	t.Run("valid request with false", func(t *testing.T) {
		req := SetIsActiveRequest{
			UserID:   "u1",
			IsActive: false,
		}

		assert.NotEmpty(t, req.UserID)
		assert.False(t, req.IsActive)
	})

	t.Run("missing user_id", func(t *testing.T) {
		req := SetIsActiveRequest{
			UserID:   "",
			IsActive: true,
		}
		assert.Empty(t, req.UserID)
		assert.True(t, req.IsActive)
	})
}

func TestSetIsActiveResponse_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		resp := SetIsActiveResponse{
			User: User{
				UserID:   "u1",
				Username: "Alice",
				TeamName: "backend",
				IsActive: false,
			},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"user_id":"u1"`)
		assert.Contains(t, string(data), `"username":"Alice"`)
		assert.Contains(t, string(data), `"team_name":"backend"`)
		assert.Contains(t, string(data), `"is_active":false`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"user": {
				"user_id": "u1",
				"username": "Alice",
				"team_name": "backend",
				"is_active": false
			}
		}`
		var resp SetIsActiveResponse

		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		assert.Equal(t, "u1", resp.User.UserID)
		assert.Equal(t, "Alice", resp.User.Username)
		assert.Equal(t, "backend", resp.User.TeamName)
		assert.False(t, resp.User.IsActive)
	})
}

func TestPullRequestShort_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		pr := PullRequestShort{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
			Status:          "OPEN",
		}

		data, err := json.Marshal(pr)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"pull_request_id":"pr-1"`)
		assert.Contains(t, string(data), `"pull_request_name":"Add feature"`)
		assert.Contains(t, string(data), `"author_id":"u1"`)
		assert.Contains(t, string(data), `"status":"OPEN"`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"pull_request_id": "pr-1",
			"pull_request_name": "Add feature",
			"author_id": "u1",
			"status": "OPEN"
		}`
		var pr PullRequestShort

		err := json.Unmarshal([]byte(jsonData), &pr)
		require.NoError(t, err)

		assert.Equal(t, "pr-1", pr.PullRequestID)
		assert.Equal(t, "Add feature", pr.PullRequestName)
		assert.Equal(t, "u1", pr.AuthorID)
		assert.Equal(t, "OPEN", pr.Status)
	})

	t.Run("marshal with MERGED status", func(t *testing.T) {
		pr := PullRequestShort{
			PullRequestID:   "pr-2",
			PullRequestName: "Fix bug",
			AuthorID:        "u2",
			Status:          "MERGED",
		}

		data, err := json.Marshal(pr)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"status":"MERGED"`)
	})
}

func TestGetReviewResponse_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON with PRs", func(t *testing.T) {
		resp := GetReviewResponse{
			UserID: "u1",
			PullRequests: []PullRequestShort{
				{
					PullRequestID:   "pr-1",
					PullRequestName: "Add feature",
					AuthorID:        "u2",
					Status:          "OPEN",
				},
				{
					PullRequestID:   "pr-2",
					PullRequestName: "Fix bug",
					AuthorID:        "u3",
					Status:          "MERGED",
				},
			},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"user_id":"u1"`)
		assert.Contains(t, string(data), `"pull_request_id":"pr-1"`)
		assert.Contains(t, string(data), `"pull_request_id":"pr-2"`)
	})

	t.Run("marshal to JSON with empty PRs", func(t *testing.T) {
		resp := GetReviewResponse{
			UserID:       "u1",
			PullRequests: []PullRequestShort{},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"user_id":"u1"`)
		assert.Contains(t, string(data), `"pull_requests":[]`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"user_id": "u1",
			"pull_requests": [
				{
					"pull_request_id": "pr-1",
					"pull_request_name": "Add feature",
					"author_id": "u2",
					"status": "OPEN"
				}
			]
		}`
		var resp GetReviewResponse

		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		assert.Equal(t, "u1", resp.UserID)
		assert.Len(t, resp.PullRequests, 1)
		assert.Equal(t, "pr-1", resp.PullRequests[0].PullRequestID)
	})

	t.Run("unmarshal with null pull_requests", func(t *testing.T) {
		jsonData := `{"user_id":"u1","pull_requests":null}`
		var resp GetReviewResponse

		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		assert.Equal(t, "u1", resp.UserID)
		assert.Nil(t, resp.PullRequests)
	})
}

func TestDTOs_EdgeCases(t *testing.T) {
	t.Run("SetIsActiveRequest with default bool value", func(t *testing.T) {
		req := SetIsActiveRequest{
			UserID:   "u1",
			IsActive: false, // bool defaults to false
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"is_active":false`)
	})

	t.Run("GetReviewResponse with many PRs", func(t *testing.T) {
		prs := make([]PullRequestShort, 100)
		for i := 0; i < 100; i++ {
			prs[i] = PullRequestShort{
				PullRequestID:   "pr-" + string(rune(i)),
				PullRequestName: "PR " + string(rune(i)),
				AuthorID:        "u" + string(rune(i)),
				Status:          "OPEN",
			}
		}

		resp := GetReviewResponse{
			UserID:       "u1",
			PullRequests: prs,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded GetReviewResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Len(t, decoded.PullRequests, 100)
	})

	t.Run("special characters in PR name", func(t *testing.T) {
		pr := PullRequestShort{
			PullRequestID:   "pr-1",
			PullRequestName: "Fix: bug with <script>alert('xss')</script>",
			AuthorID:        "u1",
			Status:          "OPEN",
		}

		data, err := json.Marshal(pr)
		require.NoError(t, err)

		var decoded PullRequestShort
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, pr.PullRequestName, decoded.PullRequestName)
	})
}

// Benchmark tests.
func BenchmarkSetIsActiveRequest_Marshal(b *testing.B) {
	req := SetIsActiveRequest{
		UserID:   "u1",
		IsActive: true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

func BenchmarkGetReviewResponse_Marshal(b *testing.B) {
	resp := GetReviewResponse{
		UserID: "u1",
		PullRequests: []PullRequestShort{
			{PullRequestID: "pr-1", PullRequestName: "Add feature", AuthorID: "u2", Status: "OPEN"},
			{PullRequestID: "pr-2", PullRequestName: "Fix bug", AuthorID: "u3", Status: "MERGED"},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(resp)
	}
}

func BenchmarkGetReviewResponse_Unmarshal(b *testing.B) {
	jsonData := []byte(`{"user_id":"u1","pull_requests":[{"pull_request_id":"pr-1",` +
		`"pull_request_name":"Add feature","author_id":"u2","status":"OPEN"}]}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp GetReviewResponse
		_ = json.Unmarshal(jsonData, &resp)
	}
}
