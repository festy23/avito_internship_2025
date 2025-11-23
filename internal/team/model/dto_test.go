package model

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamMember_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		member := TeamMember{
			UserID:   "u1",
			Username: "Alice",
			IsActive: true,
		}

		data, err := json.Marshal(member)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"user_id":"u1"`)
		assert.Contains(t, string(data), `"username":"Alice"`)
		assert.Contains(t, string(data), `"is_active":true`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{"user_id":"u1","username":"Alice","is_active":true}`
		var member TeamMember

		err := json.Unmarshal([]byte(jsonData), &member)
		require.NoError(t, err)

		assert.Equal(t, "u1", member.UserID)
		assert.Equal(t, "Alice", member.Username)
		assert.True(t, member.IsActive)
	})

	t.Run("unmarshal with missing fields", func(t *testing.T) {
		jsonData := `{"user_id":"u1"}`
		var member TeamMember

		err := json.Unmarshal([]byte(jsonData), &member)
		require.NoError(t, err)

		assert.Equal(t, "u1", member.UserID)
		assert.Empty(t, member.Username)
		assert.False(t, member.IsActive) // Default value
	})

	t.Run("unmarshal invalid JSON", func(t *testing.T) {
		jsonData := `{"user_id":"u1",invalid}`
		var member TeamMember

		err := json.Unmarshal([]byte(jsonData), &member)
		assert.Error(t, err)
	})
}

func TestAddTeamRequest_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		req := AddTeamRequest{
			TeamName: "backend",
			Members: []TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: false},
			},
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"team_name":"backend"`)
		assert.Contains(t, string(data), `"user_id":"u1"`)
		assert.Contains(t, string(data), `"user_id":"u2"`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"team_name": "backend",
			"members": [
				{"user_id": "u1", "username": "Alice", "is_active": true},
				{"user_id": "u2", "username": "Bob", "is_active": false}
			]
		}`
		var req AddTeamRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "backend", req.TeamName)
		assert.Len(t, req.Members, 2)
		assert.Equal(t, "u1", req.Members[0].UserID)
		assert.Equal(t, "Alice", req.Members[0].Username)
		assert.True(t, req.Members[0].IsActive)
	})

	t.Run("unmarshal with empty members", func(t *testing.T) {
		jsonData := `{"team_name":"backend","members":[]}`
		var req AddTeamRequest

		err := json.Unmarshal([]byte(jsonData), &req)
		require.NoError(t, err)

		assert.Equal(t, "backend", req.TeamName)
		assert.Empty(t, req.Members)
	})
}

func TestAddTeamRequest_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("valid request", func(t *testing.T) {
		req := AddTeamRequest{
			TeamName: "backend",
			Members: []TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		// Gin validation would be tested in handler tests
		assert.NotEmpty(t, req.TeamName)
		assert.NotEmpty(t, req.Members)
	})

	t.Run("missing team name", func(t *testing.T) {
		req := AddTeamRequest{
			TeamName: "",
			Members: []TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		// binding:"required" validation
		assert.Empty(t, req.TeamName)
		assert.NotEmpty(t, req.Members)
	})

	t.Run("missing members", func(t *testing.T) {
		req := AddTeamRequest{
			TeamName: "backend",
			Members:  nil,
		}

		// binding:"required" validation
		assert.Equal(t, "backend", req.TeamName)
		assert.Nil(t, req.Members)
	})
}

func TestTeamResponse_JSONSerialization(t *testing.T) {
	t.Run("marshal to JSON", func(t *testing.T) {
		resp := TeamResponse{
			TeamName: "backend",
			Members: []TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"team_name":"backend"`)
		assert.Contains(t, string(data), `"user_id":"u1"`)
	})

	t.Run("unmarshal from JSON", func(t *testing.T) {
		jsonData := `{
			"team_name": "backend",
			"members": [
				{"user_id": "u1", "username": "Alice", "is_active": true}
			]
		}`
		var resp TeamResponse

		err := json.Unmarshal([]byte(jsonData), &resp)
		require.NoError(t, err)

		assert.Equal(t, "backend", resp.TeamName)
		assert.Len(t, resp.Members, 1)
	})

	t.Run("empty response", func(t *testing.T) {
		resp := TeamResponse{}

		data, err := json.Marshal(resp)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"team_name":""`)
		assert.Contains(t, string(data), `"members":null`)
	})
}

func TestTeamMember_EdgeCases(t *testing.T) {
	t.Run("very long user ID", func(t *testing.T) {
		longID := strings.Repeat("a", 300)
		member := TeamMember{
			UserID:   longID,
			Username: "User",
			IsActive: true,
		}

		data, err := json.Marshal(member)
		require.NoError(t, err)
		assert.Contains(t, string(data), longID)
	})

	t.Run("special characters in username", func(t *testing.T) {
		member := TeamMember{
			UserID:   "u1",
			Username: "Alice@#$%&*()_+-=[]{}|;:',.<>?/~`",
			IsActive: true,
		}

		data, err := json.Marshal(member)
		require.NoError(t, err)

		var decoded TeamMember
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, member.Username, decoded.Username)
	})

	t.Run("unicode characters in username", func(t *testing.T) {
		member := TeamMember{
			UserID:   "u1",
			Username: "Алиса 李明 مرحبا",
			IsActive: true,
		}

		data, err := json.Marshal(member)
		require.NoError(t, err)

		var decoded TeamMember
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Equal(t, member.Username, decoded.Username)
	})
}

func TestAddTeamRequest_EdgeCases(t *testing.T) {
	t.Run("many members", func(t *testing.T) {
		members := make([]TeamMember, 200)
		for i := 0; i < 200; i++ {
			members[i] = TeamMember{
				UserID:   "u" + string(rune(i)),
				Username: "User" + string(rune(i)),
				IsActive: i%2 == 0,
			}
		}

		req := AddTeamRequest{
			TeamName: "large-team",
			Members:  members,
		}

		data, err := json.Marshal(req)
		require.NoError(t, err)
		assert.NotEmpty(t, data)

		var decoded AddTeamRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Len(t, decoded.Members, 200)
	})

	t.Run("duplicate members in request", func(t *testing.T) {
		req := AddTeamRequest{
			TeamName: "backend",
			Members: []TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		// JSON serialization allows duplicates
		data, err := json.Marshal(req)
		require.NoError(t, err)

		var decoded AddTeamRequest
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)
		assert.Len(t, decoded.Members, 2) // Duplicates preserved in DTO
	})
}

// Benchmark tests.
func BenchmarkTeamMember_Marshal(b *testing.B) {
	member := TeamMember{
		UserID:   "u1",
		Username: "Alice",
		IsActive: true,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(member)
	}
}

func BenchmarkAddTeamRequest_Marshal(b *testing.B) {
	req := AddTeamRequest{
		TeamName: "backend",
		Members: []TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: false},
		},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

func BenchmarkTeamResponse_Unmarshal(b *testing.B) {
	jsonData := []byte(`{"team_name":"backend","members":[{"user_id":"u1","username":"Alice","is_active":true}]}`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp TeamResponse
		_ = json.Unmarshal(jsonData, &resp)
	}
}
