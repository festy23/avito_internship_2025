package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePullRequestRequest_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		req := CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"pull_request_id":"pr-1"`)
		assert.Contains(t, string(data), `"pull_request_name":"Add feature"`)
		assert.Contains(t, string(data), `"author_id":"u1"`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"pull_request_id": "pr-1",
			"pull_request_name": "Add feature",
			"author_id": "u1"
		}`
		var req CreatePullRequestRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "pr-1", req.PullRequestID)
		assert.Equal(t, "Add feature", req.PullRequestName)
		assert.Equal(t, "u1", req.AuthorID)
	})

	t.Run("unmarshal with missing fields", func(t *testing.T) {
		jsonData := `{"pull_request_id": "pr-1"}`
		var req CreatePullRequestRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "pr-1", req.PullRequestID)
		assert.Empty(t, req.PullRequestName)
		assert.Empty(t, req.AuthorID)
	})
}

func TestCreatePullRequestRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid request", func(t *testing.T) {
		req := CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Add feature",
			AuthorID:        "u1",
		}

		assert.NotEmpty(t, req.PullRequestID)
		assert.NotEmpty(t, req.PullRequestName)
		assert.NotEmpty(t, req.AuthorID)
	})

	t.Run("missing required fields", func(t *testing.T) {
		req := CreatePullRequestRequest{}

		assert.Empty(t, req.PullRequestID)
		assert.Empty(t, req.PullRequestName)
		assert.Empty(t, req.AuthorID)
	})
}

func TestMergePullRequestRequest_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		req := MergePullRequestRequest{
			PullRequestID: "pr-1",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"pull_request_id":"pr-1"`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{"pull_request_id": "pr-1"}`
		var req MergePullRequestRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "pr-1", req.PullRequestID)
	})
}

func TestReassignReviewerRequest_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		req := ReassignReviewerRequest{
			PullRequestID: "pr-1",
			OldUserID:     "u1",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"pull_request_id":"pr-1"`)
		assert.Contains(t, string(data), `"old_user_id":"u1"`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"pull_request_id": "pr-1",
			"old_user_id": "u1"
		}`
		var req ReassignReviewerRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "pr-1", req.PullRequestID)
		assert.Equal(t, "u1", req.OldUserID)
	})

	t.Run("validation - empty fields", func(t *testing.T) {
		req := ReassignReviewerRequest{}

		assert.Empty(t, req.PullRequestID)
		assert.Empty(t, req.OldUserID)
	})
}

func TestPullRequestResponse_JSONSerialization(t *testing.T) {
	t.Run("marshal complete response", func(t *testing.T) {
		resp := PullRequestResponse{
			PullRequestID:     "pr-1",
			PullRequestName:   "Add feature",
			AuthorID:          "u1",
			Status:            StatusOPEN,
			AssignedReviewers: []string{"u2", "u3"},
			CreatedAt:         "2025-01-01T00:00:00Z",
			MergedAt:          "",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"pull_request_id":"pr-1"`)
		assert.Contains(t, string(data), `"status":"OPEN"`)
		assert.Contains(t, string(data), `"assigned_reviewers":["u2","u3"]`)
		assert.Contains(t, string(data), `"createdAt":"2025-01-01T00:00:00Z"`)
	})

	t.Run("marshal merged PR with merged_at", func(t *testing.T) {
		resp := PullRequestResponse{
			PullRequestID:     "pr-1",
			PullRequestName:   "Fix bug",
			AuthorID:          "u1",
			Status:            StatusMERGED,
			AssignedReviewers: []string{"u2"},
			CreatedAt:         "2025-01-01T00:00:00Z",
			MergedAt:          "2025-01-02T00:00:00Z",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"status":"MERGED"`)
		assert.Contains(t, string(data), `"mergedAt":"2025-01-02T00:00:00Z"`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"pull_request_id": "pr-1",
			"pull_request_name": "Add feature",
			"author_id": "u1",
			"status": "OPEN",
			"assigned_reviewers": ["u2", "u3"],
			"createdAt": "2025-01-01T00:00:00Z"
		}`
		var resp PullRequestResponse

		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		assert.Equal(t, "pr-1", resp.PullRequestID)
		assert.Equal(t, StatusOPEN, resp.Status)
		assert.Len(t, resp.AssignedReviewers, 2)
		assert.Equal(t, "2025-01-01T00:00:00Z", resp.CreatedAt)
	})

	t.Run("marshal with empty reviewers", func(t *testing.T) {
		resp := PullRequestResponse{
			PullRequestID:     "pr-1",
			PullRequestName:   "Solo PR",
			AuthorID:          "u1",
			Status:            StatusOPEN,
			AssignedReviewers: []string{},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"assigned_reviewers":[]`)
	})
}

func TestReassignReviewerResponse_JSONSerialization(t *testing.T) {
	t.Run("marshal complete response", func(t *testing.T) {
		resp := ReassignReviewerResponse{
			PR: &PullRequestResponse{
				PullRequestID:     "pr-1",
				PullRequestName:   "Add feature",
				AuthorID:          "u1",
				Status:            StatusOPEN,
				AssignedReviewers: []string{"u2", "u4"},
			},
			ReplacedBy: "u4",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"pull_request_id":"pr-1"`)
		assert.Contains(t, string(data), `"replaced_by":"u4"`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"pr": {
				"pull_request_id": "pr-1",
				"pull_request_name": "Add feature",
				"author_id": "u1",
				"status": "OPEN",
				"assigned_reviewers": ["u2", "u4"]
			},
			"replaced_by": "u4"
		}`
		var resp ReassignReviewerResponse

		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		require.NotNil(t, resp.PR)
		assert.Equal(t, "pr-1", resp.PR.PullRequestID)
		assert.Equal(t, "u4", resp.ReplacedBy)
	})
}

func TestDTOs_EdgeCases(t *testing.T) {
	t.Run("very long PR name", func(t *testing.T) {
		longName := strings.Repeat("a", 500)
		req := CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: longName,
			AuthorID:        "u1",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded CreatePullRequestRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, longName, decoded.PullRequestName)
	})

	t.Run("special characters in PR name", func(t *testing.T) {
		req := CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "Fix: bug with <script>alert('xss')</script>",
			AuthorID:        "u1",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded CreatePullRequestRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, req.PullRequestName, decoded.PullRequestName)
	})

	t.Run("unicode in PR name", func(t *testing.T) {
		req := CreatePullRequestRequest{
			PullRequestID:   "pr-1",
			PullRequestName: "修复错误: исправление бага 🚀",
			AuthorID:        "u1",
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded CreatePullRequestRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, req.PullRequestName, decoded.PullRequestName)
	})

	t.Run("many reviewers in response", func(t *testing.T) {
		reviewers := make([]string, 100)
		for i := 0; i < 100; i++ {
			reviewers[i] = "u" + string(rune(i))
		}

		resp := PullRequestResponse{
			PullRequestID:     "pr-1",
			PullRequestName:   "Big PR",
			AuthorID:          "u0",
			Status:            StatusOPEN,
			AssignedReviewers: reviewers,
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		var decoded PullRequestResponse
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Len(t, decoded.AssignedReviewers, 100)
	})

	t.Run("null PR in reassign response", func(t *testing.T) {
		resp := ReassignReviewerResponse{
			PR:         nil,
			ReplacedBy: "u2",
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)
		assert.Contains(t, string(data), `"pr":null`)
	})
}

// Benchmark tests.
func BenchmarkCreatePullRequestRequest_Marshal(b *testing.B) {
	req := CreatePullRequestRequest{
		PullRequestID:   "pr-1",
		PullRequestName: "Add feature",
		AuthorID:        "u1",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

func BenchmarkPullRequestResponse_Marshal(b *testing.B) {
	resp := PullRequestResponse{
		PullRequestID:     "pr-1",
		PullRequestName:   "Add feature",
		AuthorID:          "u1",
		Status:            StatusOPEN,
		AssignedReviewers: []string{"u2", "u3"},
		CreatedAt:         "2025-01-01T00:00:00Z",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(resp)
	}
}

func BenchmarkReassignReviewerResponse_Unmarshal(b *testing.B) {
	jsonData := []byte(`{"pr":{"pull_request_id":"pr-1","pull_request_name":"Add feature",` +
		`"author_id":"u1","status":"OPEN","assigned_reviewers":["u2","u4"]},"replaced_by":"u4"}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp ReassignReviewerResponse
		_ = json.Unmarshal(jsonData, &resp)
	}
}
